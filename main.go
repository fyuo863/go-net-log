package main

import (
	"go-net-log/internal/monitor"
)

func main() {
	// // 1. 初始化日志器
	// logger := logger.OpenLogger()
	// defer logger.Close()

	// 2. 启动监控协程
	monitor.Monitor()

}
