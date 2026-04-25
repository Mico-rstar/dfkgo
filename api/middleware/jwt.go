package middleware

import (
	"dfkgo/api/response"
	"dfkgo/auth"
	"dfkgo/errcode"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(maker auth.AuthMaker) gin.HandlerFunc {
	return func(c *gin.Context) {
		authorization := c.Request.Header.Get("Authorization")
		if authorization == "" {
			response.FailWithErr(c, errcode.ErrUnauthorized)
			c.Abort()
			return
		}
		parts := strings.SplitN(authorization, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.FailWithErr(c, errcode.ErrUnauthorized)
			c.Abort()
			return
		}
		payload, err := maker.VerifyToken(parts[1])
		if err != nil {
			response.FailWithErr(c, errcode.ErrTokenExpired)
			c.Abort()
			return
		}
		c.Set("userID", payload.UserID)
		c.Set("username", payload.Username)
		c.Next()
	}
}
