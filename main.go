package main

import (
	"dfkgo/api"
	"dfkgo/config"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	cfg := config.GetConfig()
	server := api.GetServer()

	// 启动 worker pool（异步任务处理）
	server.StartWorkers()

	// 监听系统信号，优雅关闭
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-quit
		log.Println("[main] shutting down workers...")
		server.StopWorkers()
		os.Exit(0)
	}()

	server.Start(cfg.ServerPort)
}
