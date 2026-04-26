package main

import (
	"context"
	"dfkgo/api"
	"dfkgo/config"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	cfg := config.GetConfig()
	server := api.GetServer()

	// 启动 worker pool（异步任务处理）
	server.StartWorkers()

	// 监听系统信号，优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	srv := &http.Server{
		Addr:    cfg.ServerPort,
		Handler: server.Router(),
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	log.Printf("[main] server started on %s\n", cfg.ServerPort)

	<-quit
	log.Println("[main] shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("[main] HTTP server shutdown error: %v", err)
	}

	server.StopWorkers()
	log.Println("[main] server exited")
}
