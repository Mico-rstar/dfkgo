.PHONY: server run build clean

# 启动服务器
server:
	go run main.go

# 运行项目（与 server 相同）
run: server

# 构建项目
build:
	go build -o dfkgo main.go

# 清理构建文件
clean:
	rm -f dfkgo
