package main

import (
	"fmt"
	"go-net-log/internal/logger"
	"go-net-log/internal/monitor"
)

func main() {
	// 1. 初始化日志器
	logger, err := logger.NewLogger("network_diag.jsonl")
	if err != nil {
		fmt.Printf("创建日志失败: %v\n", err)
		return
	}
	defer logger.Close()

	// 2. 启动监控协程
	monitor.Monitor(logger)

}
