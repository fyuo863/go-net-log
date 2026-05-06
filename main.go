package main

import (
	"fmt"
	"go-net-log/internal/log"
	"go-net-log/internal/net"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// 1. 初始化日志器
	logger, err := log.NewLogger("network_diag.jsonl")
	if err != nil {
		fmt.Printf("创建日志失败: %v\n", err)
		return
	}
	defer logger.Close()

	// 初始化通道
	info := net.NetInfo{
		D:         make(chan float64),
		U:         make(chan float64),
		GwResult:  make(chan net.PingResult),
		PubResult: make(chan net.PingResult),
	}

	// 启动后台协程
	go info.TrafficController()
	go info.NetworkMonitor()

	// 缓存最新数据用于展示
	var (
		curD, curU float64
		curGW      net.PingResult
		curPub     net.PingResult
	)

	fmt.Println("开始实时监控 [Ctrl+C 退出]...")
	fmt.Println("时间          下载速率     上传速率     网关延迟    公网延迟    公网丢包")
	fmt.Println("----------------------------------------------------------------------")

	// 监听退出信号
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	for {
		select {
		case <-sigChan:
			fmt.Println("\n监控已停止。")
			return
		case curD = <-info.D: //通过频率最高的通道触发记录
			_ = logger.Record(curD, curU, curGW, curPub)

		case curU = <-info.U:
		case curGW = <-info.GwResult:
		case curPub = <-info.PubResult:
		}

		// 格式化输出说明：
		// \r 回到行首
		// %-10s 等指定宽度，确保数据对齐
		// \033[K 清除当前行旧字符
		ts := time.Now().Format("15:04:05")
		fmt.Printf("\r%s | %8.2f KB/s | %8.2f KB/s | %-10v | %-10v | %.1f%%\033[K",
			ts,
			curD/1024,
			curU/1024,
			curGW.Latency.Round(time.Microsecond),
			curPub.Latency.Round(time.Microsecond),
			curPub.Loss,
		)
	}
}
