package middleware

import (
	"dfkgo/api/response"
	"dfkgo/errcode"
	"log"

	"github.com/gin-gonic/gin"
)

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("[RECOVERY] panic: %v", r)
				response.Fail(c, errcode.ErrInternal.Code, errcode.ErrInternal.Message)
				c.Abort()
			}
		}()
		c.Next()
	}
}
