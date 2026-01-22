package main

import "dfkgo/api"


func main() {
	server := api.GetServer()
	server.Start(":8888")
}