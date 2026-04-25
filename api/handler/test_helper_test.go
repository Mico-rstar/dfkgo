package handler_test

import (
	"bytes"
	"dfkgo/api"
	"dfkgo/auth"
	"dfkgo/config"
	"dfkgo/entity"
	"dfkgo/repository"
	"dfkgo/service/oss"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func setupTestServer(t *testing.T) (*api.Server, *auth.JwtMaker) {
	t.Helper()
	db, err := repository.InitTestDB()
	require.NoError(t, err)

	maker := auth.NewJwtMakerWithKey("test-secret-key-for-testing")
	cfg := config.Config{
		JwtDurationHours:      24,
		OSSBucketFiles:        "test-bucket",
		OSSBucketAvatars:      "test-avatars",
		OSSRegion:             "cn-hangzhou",
		OSSStsDurationSeconds: 900,
	}
	ossSvc := oss.NewMockOSSService()
	server := api.NewServer(db, maker, cfg, ossSvc)
	return server, maker
}

func makeAuthHeader(maker *auth.JwtMaker, userID uint64) string {
	token, _ := maker.MakeToken(userID, "test@example.com", time.Hour)
	return "Bearer " + token
}

func registerUser(t *testing.T, server *api.Server, email, password string) {
	t.Helper()
	body, _ := json.Marshal(entity.RegisterRequest{Email: email, Password: password})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Router().ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func loginUser(t *testing.T, server *api.Server, email, password string) (string, uint64) {
	t.Helper()
	body, _ := json.Marshal(entity.LoginRequest{Email: email, Password: password})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	server.Router().ServeHTTP(w, req)
	require.Equal(t, http.StatusOK, w.Code)

	var env map[string]any
	err := json.Unmarshal(w.Body.Bytes(), &env)
	require.NoError(t, err)
	data := env["data"].(map[string]any)
	token := data["token"].(string)

	// 从 token 中解析 userID
	maker := auth.NewJwtMakerWithKey("test-secret-key-for-testing")
	payload, err := maker.VerifyToken(token)
	require.NoError(t, err)

	return token, payload.UserID
}

type envelope struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func parseEnvelope(t *testing.T, w *httptest.ResponseRecorder) envelope {
	t.Helper()
	var env envelope
	err := json.Unmarshal(w.Body.Bytes(), &env)
	require.NoError(t, err)
	return env
}
