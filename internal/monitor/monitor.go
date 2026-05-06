package monitor

import (
	"fmt"
	"go-net-log/internal/fetcher"
	"go-net-log/internal/logger"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// 监控核心逻辑，负责协调数据采集和日志记录
type Temp struct {
	curD, curU float64
	curGW      fetcher.PingResult
	curPub     fetcher.PingResult
}

func Monitor(logger *logger.Logger) {

	// 初始化通道
	info := fetcher.NetInfo{
		D:         make(chan float64),
		U:         make(chan float64),
		GwResult:  make(chan fetcher.PingResult),
		PubResult: make(chan fetcher.PingResult),
	}
	// 缓存最新数据用于展示
	var Temp = &Temp{}

	go info.TrafficController()
	go info.NetworkMonitor()
	go Temp.UIUpdate()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	fmt.Println("时间          下载速率     上传速率     网关延迟    公网延迟    公网丢包")
	fmt.Println("----------------------------------------------------------------------")
	for {
		select {
		case <-sigChan:
			fmt.Println("\n监控已停止。")
			return
		case Temp.curD = <-info.D: //通过频率最高的通道触发记录
			_ = logger.Record(Temp.curD, Temp.curU, Temp.curGW, Temp.curPub)
			Temp.UIUpdate()
		case Temp.curU = <-info.U:
		case Temp.curGW = <-info.GwResult:
		case Temp.curPub = <-info.PubResult:
		}
	}

}

func (t *Temp) UIUpdate() {
	ts := time.Now().Format("15:04:05")
	fmt.Printf("\r%s | %8.2f KB/s | %8.2f KB/s | %-10v | %-10v | %.1f%%\033[K",
		ts,
		t.curD/1024,
		t.curU/1024,
		t.curGW.Latency.Round(time.Microsecond),
		t.curPub.Latency.Round(time.Microsecond),
		t.curPub.Loss,
	)
}
