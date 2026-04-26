package handler

import (
	"bytes"
	"dfkgo/api/middleware"
	"dfkgo/auth"
	"dfkgo/model"
	"dfkgo/repository"
	taskservice "dfkgo/service/task"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func setupHistoryTestRouter(t *testing.T) (*gin.Engine, *repository.FileRepo, string) {
	gin.SetMode(gin.TestMode)

	db, err := repository.InitTestDB()
	require.NoError(t, err)

	maker := auth.NewJwtMakerWithKey("test-secret-key-for-testing-only")
	taskRepo := repository.NewTaskRepo(db)
	fileRepo := repository.NewFileRepo(db)
	queue := taskservice.NewMemoryQueue(100)
	taskSvc := taskservice.NewTaskService(taskRepo, fileRepo, queue)

	token, err := maker.MakeToken(1, "testuser", 24*time.Hour)
	require.NoError(t, err)

	router := gin.New()
	api := router.Group("/api")
	protected := api.Group("")
	protected.Use(middleware.AuthMiddleware(maker))

	// 注册 task 路由（创建 task 用）
	taskHandler := NewTaskHandler(taskSvc)
	taskGroup := protected.Group("/tasks")
	taskGroup.POST("", taskHandler.CreateTask)

	// 注册 history 路由
	historyHandler := NewHistoryHandler(taskSvc)
	historyGroup := protected.Group("/history")
	historyGroup.GET("", historyHandler.ListHistory)
	historyGroup.DELETE("/:taskId", historyHandler.DeleteHistory)
	historyGroup.POST("/batch-delete", historyHandler.BatchDeleteHistory)
	historyGroup.GET("/stats", historyHandler.GetStats)

	return router, fileRepo, token
}

func createHistoryTestFile(t *testing.T, fileRepo *repository.FileRepo) {
	file := &model.File{
		FileUID:      "file_hist001",
		UserID:       1,
		FileName:     "history_test.jpg",
		MimeType:     "image/jpeg",
		FileSize:     2048,
		Modality:     "image",
		MD5:          "e41d8cd98f00b204e9800998ecf8427e",
		OSSBucket:    "test-bucket",
		OSSObjectKey: "files/1/history_test.jpg",
		OSSURL:       "https://oss.example.com/files/1/history_test.jpg",
		UploadStatus: "completed",
	}
	require.NoError(t, fileRepo.Create(file))
}

func createTaskViaAPI(t *testing.T, router *gin.Engine, token string) string {
	body, _ := json.Marshal(map[string]string{
		"fileId":   "file_hist001",
		"modality": "image",
	})
	req := httptest.NewRequest("POST", "/api/tasks", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	return resp["data"].(map[string]any)["taskId"].(string)
}

func TestHistoryHandler_ListHistory(t *testing.T) {
	router, fileRepo, token := setupHistoryTestRouter(t)
	createHistoryTestFile(t, fileRepo)
	createTaskViaAPI(t, router, token)

	req := httptest.NewRequest("GET", "/api/history?page=1&limit=10", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	require.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]any)
	items := data["items"].([]any)
	require.Len(t, items, 1)

	item := items[0].(map[string]any)
	require.Equal(t, "history_test.jpg", item["fileName"])
	require.Equal(t, float64(2048), item["fileSize"])

	pagination := data["pagination"].(map[string]any)
	require.Equal(t, float64(1), pagination["total"])
}

func TestHistoryHandler_DeleteHistory(t *testing.T) {
	router, fileRepo, token := setupHistoryTestRouter(t)
	createHistoryTestFile(t, fileRepo)
	taskID := createTaskViaAPI(t, router, token)

	req := httptest.NewRequest("DELETE", "/api/history/"+taskID, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	require.Equal(t, float64(0), resp["code"])

	// 验证已删除
	req2 := httptest.NewRequest("GET", "/api/history?page=1&limit=10", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	var resp2 map[string]any
	json.Unmarshal(w2.Body.Bytes(), &resp2)
	data := resp2["data"].(map[string]any)
	items := data["items"].([]any)
	require.Len(t, items, 0)
}

func TestHistoryHandler_BatchDeleteHistory(t *testing.T) {
	router, fileRepo, token := setupHistoryTestRouter(t)
	createHistoryTestFile(t, fileRepo)
	taskID := createTaskViaAPI(t, router, token)

	body, _ := json.Marshal(map[string]any{
		"taskIds": []string{taskID},
	})
	req := httptest.NewRequest("POST", "/api/history/batch-delete", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	require.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]any)
	require.Equal(t, float64(1), data["deletedCount"])
}

func TestHistoryHandler_GetStats(t *testing.T) {
	router, fileRepo, token := setupHistoryTestRouter(t)
	createHistoryTestFile(t, fileRepo)
	createTaskViaAPI(t, router, token)

	req := httptest.NewRequest("GET", "/api/history/stats", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	require.Equal(t, float64(0), resp["code"])

	data := resp["data"].(map[string]any)
	require.NotNil(t, data["byModality"])
	require.NotNil(t, data["byCategory"])
}
