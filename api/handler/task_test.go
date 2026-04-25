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

func setupTaskTestRouter(t *testing.T) (*gin.Engine, *repository.FileRepo, string) {
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

	taskHandler := NewTaskHandler(taskSvc)
	taskGroup := protected.Group("/tasks")
	taskGroup.POST("", taskHandler.CreateTask)
	taskGroup.GET("/:taskId", taskHandler.GetTaskStatus)
	taskGroup.GET("/:taskId/result", taskHandler.GetTaskResult)
	taskGroup.POST("/:taskId/cancel", taskHandler.CancelTask)

	return router, fileRepo, token
}

func createTestFileForHandler(t *testing.T, fileRepo *repository.FileRepo) {
	file := &model.File{
		FileUID:      "file_abc123",
		UserID:       1,
		FileName:     "test.jpg",
		MimeType:     "image/jpeg",
		FileSize:     1024,
		Modality:     "image",
		MD5:          "d41d8cd98f00b204e9800998ecf8427e",
		OSSBucket:    "test-bucket",
		OSSObjectKey: "files/1/test.jpg",
		OSSURL:       "https://oss.example.com/files/1/test.jpg",
		UploadStatus: "completed",
	}
	require.NoError(t, fileRepo.Create(file))
}

func TestTaskHandler_CreateTask(t *testing.T) {
	router, fileRepo, token := setupTaskTestRouter(t)
	createTestFileForHandler(t, fileRepo)

	body, _ := json.Marshal(map[string]string{
		"fileId":   "file_abc123",
		"modality": "image",
	})
	req := httptest.NewRequest("POST", "/api/tasks", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.Equal(t, float64(0), resp["code"])
	data := resp["data"].(map[string]any)
	require.Contains(t, data["taskId"], "task_")
	require.Equal(t, "pending", data["status"])
}

func TestTaskHandler_CreateTask_MissingFields(t *testing.T) {
	router, _, token := setupTaskTestRouter(t)

	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest("POST", "/api/tasks", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	require.NotEqual(t, float64(0), resp["code"])
}

func TestTaskHandler_GetTaskStatus(t *testing.T) {
	router, fileRepo, token := setupTaskTestRouter(t)
	createTestFileForHandler(t, fileRepo)

	// 先创建 task
	body, _ := json.Marshal(map[string]string{
		"fileId":   "file_abc123",
		"modality": "image",
	})
	req := httptest.NewRequest("POST", "/api/tasks", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var createResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &createResp)
	taskID := createResp["data"].(map[string]any)["taskId"].(string)

	// 查询状态
	req2 := httptest.NewRequest("GET", "/api/tasks/"+taskID, nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	require.Equal(t, http.StatusOK, w2.Code)
	var resp map[string]any
	json.Unmarshal(w2.Body.Bytes(), &resp)
	require.Equal(t, float64(0), resp["code"])
	data := resp["data"].(map[string]any)
	require.Equal(t, taskID, data["taskId"])
	require.Equal(t, "pending", data["status"])
}

func TestTaskHandler_CancelTask(t *testing.T) {
	router, fileRepo, token := setupTaskTestRouter(t)
	createTestFileForHandler(t, fileRepo)

	// 创建 task
	body, _ := json.Marshal(map[string]string{
		"fileId":   "file_abc123",
		"modality": "image",
	})
	req := httptest.NewRequest("POST", "/api/tasks", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var createResp map[string]any
	json.Unmarshal(w.Body.Bytes(), &createResp)
	taskID := createResp["data"].(map[string]any)["taskId"].(string)

	// 取消
	req2 := httptest.NewRequest("POST", "/api/tasks/"+taskID+"/cancel", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	require.Equal(t, http.StatusOK, w2.Code)
	var resp map[string]any
	json.Unmarshal(w2.Body.Bytes(), &resp)
	require.Equal(t, float64(0), resp["code"])
}

func TestTaskHandler_Unauthorized(t *testing.T) {
	router, _, _ := setupTaskTestRouter(t)

	req := httptest.NewRequest("GET", "/api/tasks/task_xxx", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	var resp map[string]any
	json.Unmarshal(w.Body.Bytes(), &resp)
	require.NotEqual(t, float64(0), resp["code"])
}
