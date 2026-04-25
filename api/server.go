package api

import (
	"dfkgo/api/handler"
	"dfkgo/api/middleware"
	"dfkgo/auth"
	"dfkgo/config"
	"dfkgo/repository"
	"sync"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Server struct {
	router   *gin.Engine
	maker    auth.AuthMaker
	db       *gorm.DB
	config   config.Config
	userRepo *repository.UserRepo
	fileRepo *repository.FileRepo
	taskRepo *repository.TaskRepo
}

var (
	server *Server
	once   sync.Once
)

func GetServer() *Server {
	once.Do(func() {
		server = buildServer()
	})
	return server
}

// NewServer 用于测试注入
func NewServer(db *gorm.DB, maker auth.AuthMaker, cfg config.Config) *Server {
	s := &Server{
		maker:    maker,
		db:       db,
		config:   cfg,
		userRepo: repository.NewUserRepo(db),
		fileRepo: repository.NewFileRepo(db),
		taskRepo: repository.NewTaskRepo(db),
	}
	s.setupRoutes()
	return s
}

func buildServer() *Server {
	cfg := config.GetConfig()
	maker := auth.NewJwtMaker()
	db, err := repository.InitDB(cfg.DBDriver, cfg.DBSource)
	if err != nil {
		panic("failed to connect database: " + err.Error())
	}
	return NewServer(db, maker, cfg)
}

func (s *Server) setupRoutes() {
	router := gin.Default()
	router.Use(middleware.Recovery())

	api := router.Group("/api")
	{
		// 公开路由
		_ = api.Group("/auth")
		// Phase 2A: authGroup.POST("/register", ...)
		// Phase 2A: authGroup.POST("/login", ...)

		// 鉴权路由
		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware(s.maker))
		{
			h := handler.NewHealthHandler()
			protected.GET("/health", h.Health)

			// 用户域 - Phase 2A
			// userGroup := protected.Group("/user")

			// 文件域 - Phase 2A
			// uploadGroup := protected.Group("/upload")

			// 任务域 - Phase 2B
			// taskGroup := protected.Group("/tasks")

			// 历史域 - Phase 2B
			// historyGroup := protected.Group("/history")
		}
	}

	s.router = router
}

func (s *Server) Start(address string) error {
	return s.router.Run(address)
}

func (s *Server) Router() *gin.Engine {
	return s.router
}

func (s *Server) DB() *gorm.DB {
	return s.db
}
