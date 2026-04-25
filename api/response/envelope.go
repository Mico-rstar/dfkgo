package response

import (
	"dfkgo/errcode"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type Envelope struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	Data      any    `json:"data"`
	Timestamp int64  `json:"timestamp"`
}

func OK(c *gin.Context, data any) {
	c.JSON(http.StatusOK, Envelope{
		Code:      0,
		Message:   "Success",
		Data:      data,
		Timestamp: time.Now().Unix(),
	})
}

func OKWithMsg(c *gin.Context, msg string, data any) {
	c.JSON(http.StatusOK, Envelope{
		Code:      0,
		Message:   msg,
		Data:      data,
		Timestamp: time.Now().Unix(),
	})
}

func Fail(c *gin.Context, code int, msg string) {
	c.JSON(http.StatusOK, Envelope{
		Code:      code,
		Message:   msg,
		Data:      nil,
		Timestamp: time.Now().Unix(),
	})
}

func FailWithErr(c *gin.Context, err error) {
	if e, ok := errcode.IsErrCode(err); ok {
		Fail(c, e.Code, e.Message)
		return
	}
	Fail(c, errcode.ErrInternal.Code, errcode.ErrInternal.Message)
}
