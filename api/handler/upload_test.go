package handler_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"dfkgo/entity"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUploadInit(t *testing.T) {
	server, _ := setupTestServer(t)
	registerUser(t, server, "upload@example.com", "password123")
	token, _ := loginUser(t, server, "upload@example.com", "password123")

	body, _ := json.Marshal(entity.UploadInitRequest{
		FileName: "video.mp4",
		FileSize: 1024 * 1024,
		MimeType: "video/mp4",
		MD5:      "9e107d9d372bb6826bd81d3542a419d6",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/upload/init", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	server.Router().ServeHTTP(w, req)

	env := parseEnvelope(t, w)
	require.Equal(t, 0, env.Code)

	var data entity.UploadInitResponse
	err := json.Unmarshal(env.Data, &data)
	require.NoError(t, err)
	assert.False(t, data.Hit)
	assert.NotEmpty(t, data.FileId)
	assert.NotNil(t, data.STS)
	assert.NotEmpty(t, data.STS.AccessKeyID)
}

func TestUploadInitDuplicate(t *testing.T) {
	server, _ := setupTestServer(t)
	registerUser(t, server, "dup-upload@example.com", "password123")
	token, _ := loginUser(t, server, "dup-upload@example.com", "password123")

	initReq := entity.UploadInitRequest{
		FileName: "video.mp4",
		FileSize: 1024 * 1024,
		MimeType: "video/mp4",
		MD5:      "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1",
	}

	// 第一次上传
	body, _ := json.Marshal(initReq)
	req := httptest.NewRequest(http.MethodPost, "/api/upload/init", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	server.Router().ServeHTTP(w, req)
	env := parseEnvelope(t, w)
	require.Equal(t, 0, env.Code)

	var firstResp entity.UploadInitResponse
	_ = json.Unmarshal(env.Data, &firstResp)

	// 完成回调
	cbBody, _ := json.Marshal(entity.UploadCallbackRequest{FileId: firstResp.FileId})
	cbReq := httptest.NewRequest(http.MethodPost, "/api/upload/callback", bytes.NewReader(cbBody))
	cbReq.Header.Set("Content-Type", "application/json")
	cbReq.Header.Set("Authorization", "Bearer "+token)
	cbW := httptest.NewRecorder()
	server.Router().ServeHTTP(cbW, cbReq)
	cbEnv := parseEnvelope(t, cbW)
	require.Equal(t, 0, cbEnv.Code)

	// 第二次相同 MD5 → 秒传命中
	body2, _ := json.Marshal(initReq)
	req2 := httptest.NewRequest(http.MethodPost, "/api/upload/init", bytes.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	server.Router().ServeHTTP(w2, req2)

	env2 := parseEnvelope(t, w2)
	require.Equal(t, 0, env2.Code)

	var secondResp entity.UploadInitResponse
	_ = json.Unmarshal(env2.Data, &secondResp)
	assert.True(t, secondResp.Hit)
	assert.NotEmpty(t, secondResp.OssUrl)
	assert.Equal(t, firstResp.FileId, secondResp.FileId)
}

func TestUploadCallback(t *testing.T) {
	server, _ := setupTestServer(t)
	registerUser(t, server, "callback@example.com", "password123")
	token, _ := loginUser(t, server, "callback@example.com", "password123")

	// 先 init
	body, _ := json.Marshal(entity.UploadInitRequest{
		FileName: "image.png",
		FileSize: 512 * 1024,
		MimeType: "image/png",
		MD5:      "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/upload/init", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	server.Router().ServeHTTP(w, req)
	env := parseEnvelope(t, w)
	require.Equal(t, 0, env.Code)

	var initResp entity.UploadInitResponse
	_ = json.Unmarshal(env.Data, &initResp)

	// callback
	cbBody, _ := json.Marshal(entity.UploadCallbackRequest{FileId: initResp.FileId})
	cbReq := httptest.NewRequest(http.MethodPost, "/api/upload/callback", bytes.NewReader(cbBody))
	cbReq.Header.Set("Content-Type", "application/json")
	cbReq.Header.Set("Authorization", "Bearer "+token)
	cbW := httptest.NewRecorder()
	server.Router().ServeHTTP(cbW, cbReq)

	cbEnv := parseEnvelope(t, cbW)
	assert.Equal(t, 0, cbEnv.Code)
	assert.Equal(t, "文件上传完成", cbEnv.Message)

	var cbResp entity.UploadCallbackResponse
	_ = json.Unmarshal(cbEnv.Data, &cbResp)
	assert.Equal(t, initResp.FileId, cbResp.FileId)
	assert.NotEmpty(t, cbResp.OssUrl)
}
