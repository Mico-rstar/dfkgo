package response

import (
	"dfkgo/errcode"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestOK(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	OK(c, gin.H{"key": "value"})

	assert.Equal(t, http.StatusOK, w.Code)
	var env Envelope
	err := json.Unmarshal(w.Body.Bytes(), &env)
	assert.NoError(t, err)
	assert.Equal(t, 0, env.Code)
	assert.Equal(t, "Success", env.Message)
	assert.NotNil(t, env.Data)
	assert.NotZero(t, env.Timestamp)
}

func TestFail(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Fail(c, 401005, "用户密码错误，请重新输入")

	assert.Equal(t, http.StatusOK, w.Code)
	var env Envelope
	err := json.Unmarshal(w.Body.Bytes(), &env)
	assert.NoError(t, err)
	assert.Equal(t, 401005, env.Code)
	assert.Equal(t, "用户密码错误，请重新输入", env.Message)
	assert.Nil(t, env.Data)
}

func TestFailWithErr_ErrCode(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	FailWithErr(c, errcode.ErrWrongPassword)

	assert.Equal(t, http.StatusOK, w.Code)
	var env Envelope
	err := json.Unmarshal(w.Body.Bytes(), &env)
	assert.NoError(t, err)
	assert.Equal(t, errcode.ErrWrongPassword.Code, env.Code)
	assert.Equal(t, errcode.ErrWrongPassword.Message, env.Message)
}

func TestFailWithErr_GenericError(t *testing.T) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	FailWithErr(c, errors.New("some unknown error"))

	assert.Equal(t, http.StatusOK, w.Code)
	var env Envelope
	err := json.Unmarshal(w.Body.Bytes(), &env)
	assert.NoError(t, err)
	assert.Equal(t, errcode.ErrInternal.Code, env.Code)
	assert.Equal(t, errcode.ErrInternal.Message, env.Message)
}
