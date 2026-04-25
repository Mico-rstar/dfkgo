package api

import (
	"dfkgo/api/handler"
	"dfkgo/api/middleware"
	"dfkgo/auth"
	"dfkgo/config"
	"dfkgo/repository"
	authsvc "dfkgo/service/auth"
	filesvc "dfkgo/service/file"
	"dfkgo/service/oss"
	taskservice "dfkgo/service/task"
	usersvc "dfkgo/service/user"
	"sync"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type Server struct {
	router      *gin.Engine
	maker       auth.AuthMaker
	db          *gorm.DB
	config      config.Config
	ossService  oss.OSSService
	userRepo    *repository.UserRepo
	fileRepo    *repository.FileRepo
	taskRepo    *repository.TaskRepo
	queue       taskservice.TaskQueue
	workerPool  *taskservice.WorkerPool
	taskService *taskservice.TaskService
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
func NewServer(db *gorm.DB, maker auth.AuthMaker, cfg config.Config, ossService oss.OSSService) *Server {
	userRepo := repository.NewUserRepo(db)
	fileRepo := repository.NewFileRepo(db)
	taskRepo := repository.NewTaskRepo(db)

	queueCap := cfg.TaskQueueCapacity
	if queueCap <= 0 {
		queueCap = 1000
	}
	queue := taskservice.NewMemoryQueue(queueCap)
	taskSvc := taskservice.NewTaskService(taskRepo, fileRepo, queue)

	poolSize := cfg.TaskWorkerPoolSize
	if poolSize <= 0 {
		poolSize = 4
	}
	modelClient := taskservice.NewHTTPModelClient(cfg.ModelServerBaseURL, cfg.ModelServerTimeoutSec)
	workerPool := taskservice.NewWorkerPool(queue, taskRepo, fileRepo, modelClient, poolSize)

	s := &Server{
		maker:       maker,
		db:          db,
		config:      cfg,
		ossService:  ossService,
		userRepo:    userRepo,
		fileRepo:    fileRepo,
		taskRepo:    taskRepo,
		queue:       queue,
		workerPool:  workerPool,
		taskService: taskSvc,
	}
	s.setupRoutes()
	return s
}

// NewServerWithDeps 用于测试注入自定义依赖
func NewServerWithDeps(db *gorm.DB, maker auth.AuthMaker, cfg config.Config, ossService oss.OSSService, queue taskservice.TaskQueue, client taskservice.ModelClient) *Server {
	userRepo := repository.NewUserRepo(db)
	fileRepo := repository.NewFileRepo(db)
	taskRepo := repository.NewTaskRepo(db)

	taskSvc := taskservice.NewTaskService(taskRepo, fileRepo, queue)

	poolSize := cfg.TaskWorkerPoolSize
	if poolSize <= 0 {
		poolSize = 4
	}
	workerPool := taskservice.NewWorkerPool(queue, taskRepo, fileRepo, client, poolSize)

	s := &Server{
		maker:       maker,
		db:          db,
		config:      cfg,
		ossService:  ossService,
		userRepo:    userRepo,
		fileRepo:    fileRepo,
		taskRepo:    taskRepo,
		queue:       queue,
		workerPool:  workerPool,
		taskService: taskSvc,
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
	ossSvc, err := oss.NewOSSService(cfg)
	if err != nil {
		panic("failed to init OSS service: " + err.Error())
	}
	return NewServer(db, maker, cfg, ossSvc)
}

func (s *Server) setupRoutes() {
	router := gin.Default()
	router.Use(middleware.Recovery())

	api := router.Group("/api")
	{
		// 公开路由
		authService := authsvc.NewAuthService(s.userRepo, s.maker, s.config)
		authHandler := handler.NewAuthHandler(authService)
		authGroup := api.Group("/auth")
		authGroup.POST("/register", authHandler.Register)
		authGroup.POST("/login", authHandler.Login)

		// 鉴权路由
		protected := api.Group("")
		protected.Use(middleware.AuthMiddleware(s.maker))
		{
			h := handler.NewHealthHandler()
			protected.GET("/health", h.Health)

			// 用户域
			userService := usersvc.NewUserService(s.userRepo, s.ossService, s.config)
			userHandler := handler.NewUserHandler(userService)
			userGroup := protected.Group("/user")
			userGroup.GET("/get-profile", userHandler.GetProfile)
			userGroup.PUT("/update-profile", userHandler.UpdateProfile)
			userGroup.POST("/avatar-upload/init", userHandler.InitAvatarUpload)
			userGroup.POST("/avatar-upload/callback", userHandler.AvatarUploadCallback)
			userGroup.GET("/fetch-avatar", userHandler.FetchAvatar)

			// 文件域
			fileService := filesvc.NewFileService(s.fileRepo, s.ossService, s.config)
			uploadHandler := handler.NewUploadHandler(fileService)
			uploadGroup := protected.Group("/upload")
			uploadGroup.POST("/init", uploadHandler.InitUpload)
			uploadGroup.POST("/callback", uploadHandler.UploadCallback)

			// 任务域
			taskGroup := protected.Group("/tasks")
			taskHandler := handler.NewTaskHandler(s.taskService)
			taskGroup.POST("", taskHandler.CreateTask)
			taskGroup.GET("/:taskId", taskHandler.GetTaskStatus)
			taskGroup.GET("/:taskId/result", taskHandler.GetTaskResult)
			taskGroup.POST("/:taskId/cancel", taskHandler.CancelTask)

			// 历史域
			historyGroup := protected.Group("/history")
			historyHandler := handler.NewHistoryHandler(s.taskService, s.fileRepo)
			historyGroup.GET("", historyHandler.ListHistory)
			historyGroup.DELETE("/:taskId", historyHandler.DeleteHistory)
			historyGroup.POST("/batch-delete", historyHandler.BatchDeleteHistory)
			historyGroup.GET("/stats", historyHandler.GetStats)
		}
	}

	s.router = router
}

func (s *Server) StartWorkers() {
	s.taskService.RecoverOrphanTasks()
	s.workerPool.Start()
}

func (s *Server) StopWorkers() {
	s.workerPool.Stop()
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
