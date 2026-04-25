package main

import (
	"dfkgo/api"
	"dfkgo/config"
)

func main() {
	cfg := config.GetConfig()
	server := api.GetServer()
	server.Start(cfg.ServerPort)
}
