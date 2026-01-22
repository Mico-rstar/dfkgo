package api

import (
	"github.com/gin-gonic/gin"
)

func (server *Server) Health(c *gin.Context) {
	c.JSON(200, gin.H{
		"status": "ok",
	})
}
