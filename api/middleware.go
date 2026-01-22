package api

import (
	"dfkgo/auth"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware(maker auth.AuthMaker) func(*gin.Context) {
	return func(ctx *gin.Context) {
		authorization := ctx.Request.Header.Get("Authorization")
		if authorization == "" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Unauthorized"})
			return
		}
		parts := strings.SplitN(authorization, " ", 2)
		if len(parts) != 2 {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Invalid authoriazation format"})
			return
		}
		if parts[0] != "Bearer" {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Unsupported authorization schema"})
			return
		}
		// Verify jwt token
		jwtToken := parts[1]
		// maker := auth.NewJwtMaker()
		payload, err := maker.VerifyToken(jwtToken)
		if err != nil {
			ctx.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"mesage": err.Error()})
			return
		}
		fmt.Println(payload)
		ctx.Next()
	}

}
