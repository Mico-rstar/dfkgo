package api

import (
	"dfkgo/auth"
	"sync"

	"github.com/gin-gonic/gin"
)

type Server struct {
	router *gin.Engine
	maker  auth.AuthMaker
}

var (
	server *Server
	// authService *AuthService
	once sync.Once
)

// func GetAuthService() *AuthService {
// 	once.Do(func() {
// 		service = &service.NewAuthService()
// 	})
// 	return service
// }

func GetServer() *Server {
	once.Do(func() {
		server = buildServer()
	})
	return server
}

// factory function for server
func buildServer() *Server {
	maker := auth.NewJwtMaker()
	server = &Server{
		maker: maker,
	}
	server.setupServer()
	return server

}

func (s *Server) setupServer() {
	router := gin.Default()
	router.POST("/auth/register", s.Register)
	router.POST("/auth/login", s.Login)

	authRouter := router.Group("/").Use(AuthMiddleware(s.maker))
	authRouter.GET("/health", s.Health)

	s.router = router
}

func (s *Server) Start(address string) error {
	return s.router.Run(address)
}
