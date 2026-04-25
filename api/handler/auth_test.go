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

func TestRegisterSuccess(t *testing.T) {
	server, _ := setupTestServer(t)
	body, _ := json.Marshal(entity.RegisterRequest{Email: "user@example.com", Password: "password123"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Router().ServeHTTP(w, req)

	env := parseEnvelope(t, w)
	assert.Equal(t, 0, env.Code)
	assert.Equal(t, "注册成功", env.Message)
}

func TestRegisterDuplicateEmail(t *testing.T) {
	server, _ := setupTestServer(t)
	email := "dup@example.com"
	registerUser(t, server, email, "password123")

	body, _ := json.Marshal(entity.RegisterRequest{Email: email, Password: "password456"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Router().ServeHTTP(w, req)

	env := parseEnvelope(t, w)
	assert.Equal(t, 401009, env.Code)
}

func TestRegisterInvalidEmail(t *testing.T) {
	server, _ := setupTestServer(t)
	body, _ := json.Marshal(map[string]string{"email": "not-an-email", "password": "password123"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Router().ServeHTTP(w, req)

	env := parseEnvelope(t, w)
	assert.NotEqual(t, 0, env.Code)
}

func TestLoginSuccess(t *testing.T) {
	server, _ := setupTestServer(t)
	registerUser(t, server, "login@example.com", "password123")

	body, _ := json.Marshal(entity.LoginRequest{Email: "login@example.com", Password: "password123"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Router().ServeHTTP(w, req)

	env := parseEnvelope(t, w)
	require.Equal(t, 0, env.Code)

	var data entity.LoginResponse
	err := json.Unmarshal(env.Data, &data)
	require.NoError(t, err)
	assert.NotEmpty(t, data.Token)
	assert.Greater(t, data.ExpiresAt, int64(0))
}

func TestLoginWrongPassword(t *testing.T) {
	server, _ := setupTestServer(t)
	registerUser(t, server, "wrong@example.com", "password123")

	body, _ := json.Marshal(entity.LoginRequest{Email: "wrong@example.com", Password: "wrongpassword"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Router().ServeHTTP(w, req)

	env := parseEnvelope(t, w)
	assert.Equal(t, 401005, env.Code)
}

func TestLoginUserNotFound(t *testing.T) {
	server, _ := setupTestServer(t)
	body, _ := json.Marshal(entity.LoginRequest{Email: "noone@example.com", Password: "password123"})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Router().ServeHTTP(w, req)

	env := parseEnvelope(t, w)
	assert.Equal(t, 401004, env.Code)
}
