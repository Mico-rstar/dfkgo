package api

import (
	"sync"

	"github.com/gin-gonic/gin"
)

type Server struct {
	router *gin.Engine
}

var (
	server *Server
	authService *AuthService
	once   sync.Once
)

func GetAuthService() *AuthService {
	once.Do(func() {
		service = &service.NewAuthService()
	})
	return service
}

func GetServer() *Server {
	once.Do(func() {
		server = buildServer()
	})
	return server
}

// factory function for server
func buildServer() *Server {
	router := gin.Default()
	router.GET("/health", Health)
	router.POST("/register", Register)
	router.POST("/login", Login)

	return &Server{
		router: router,
	}
}

func (s *Server) Start(address string) error {
	return s.router.Run(address)
}
