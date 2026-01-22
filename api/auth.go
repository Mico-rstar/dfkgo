package api

import (
	"dfkgo/auth"
	"dfkgo/entity"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

func (server *Server) Register(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "ok",
	})
}

func (server *Server) Login(c *gin.Context) {
	// mock
	const email = "2114678406@qq.com"
	const username = "2114678406@qq.com"
	const password = "1237789ab"

	loginInfo := entity.EmlPswLoginInfo{}
	err := c.ShouldBindBodyWithJSON(&loginInfo)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": err.Error(),
		})
		return
	}
	// check login info
	if loginInfo.Email != email || loginInfo.Password != password {
		c.JSON(http.StatusUnauthorized, gin.H{
			"message": "Error email or password",
		})
	}

	maker := auth.NewJwtMaker()
	token, err := maker.MakeToken(username, time.Hour)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusAccepted, gin.H{
		"token":  token,
	})
}
