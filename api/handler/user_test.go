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

func TestGetProfile(t *testing.T) {
	server, _ := setupTestServer(t)
	registerUser(t, server, "profile@example.com", "password123")
	token, _ := loginUser(t, server, "profile@example.com", "password123")

	req := httptest.NewRequest(http.MethodGet, "/api/user/get-profile", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	server.Router().ServeHTTP(w, req)

	env := parseEnvelope(t, w)
	require.Equal(t, 0, env.Code)

	var data entity.ProfileResponse
	err := json.Unmarshal(env.Data, &data)
	require.NoError(t, err)
	assert.Equal(t, "profile@example.com", data.Email)
}

func TestUpdateProfile(t *testing.T) {
	server, _ := setupTestServer(t)
	registerUser(t, server, "update@example.com", "password123")
	token, _ := loginUser(t, server, "update@example.com", "password123")

	nickname := "新昵称"
	body, _ := json.Marshal(entity.UpdateProfileRequest{Nickname: &nickname})
	req := httptest.NewRequest(http.MethodPut, "/api/user/update-profile", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	server.Router().ServeHTTP(w, req)

	env := parseEnvelope(t, w)
	assert.Equal(t, 0, env.Code)
	assert.Equal(t, "用户信息更新成功", env.Message)

	// 验证更新生效
	req2 := httptest.NewRequest(http.MethodGet, "/api/user/get-profile", nil)
	req2.Header.Set("Authorization", "Bearer "+token)
	w2 := httptest.NewRecorder()
	server.Router().ServeHTTP(w2, req2)

	env2 := parseEnvelope(t, w2)
	var data entity.ProfileResponse
	_ = json.Unmarshal(env2.Data, &data)
	assert.Equal(t, "新昵称", data.Nickname)
}

func TestFetchAvatar(t *testing.T) {
	server, _ := setupTestServer(t)
	registerUser(t, server, "avatar@example.com", "password123")
	token, _ := loginUser(t, server, "avatar@example.com", "password123")

	req := httptest.NewRequest(http.MethodGet, "/api/user/fetch-avatar", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	server.Router().ServeHTTP(w, req)

	env := parseEnvelope(t, w)
	require.Equal(t, 0, env.Code)

	var data entity.FetchAvatarResponse
	err := json.Unmarshal(env.Data, &data)
	require.NoError(t, err)
	// 新注册用户头像为空
	assert.Equal(t, "", data.AvatarUrl)
}
