package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-ping/ping"
	"github.com/jackpal/gateway"
	"github.com/shirou/gopsutil/v4/net"
)

// PingResult 将延迟和丢包打包，防止 UI 显示不同步
type PingResult struct {
	Latency time.Duration
	Loss    float64
}

// NetInfo 结构体定义
type NetInfo struct {
	d         chan float64    // 下载速率 (Bytes/s)
	u         chan float64    // 上传速率 (Bytes/s)
	gwResult  chan PingResult // 网关质量数据
	pubResult chan PingResult // 公网质量数据
}

// 获取网卡总流量
func getTotalBytes() (uint64, uint64, error) {
	counters, err := net.IOCounters(false)
	if err != nil || len(counters) == 0 {
		return 0, 0, err
	}
	// counters[0] 是所有网卡的聚合统计
	return counters[0].BytesRecv, counters[0].BytesSent, nil
}

// 流量采样协程
func (n *NetInfo) trafficController() {
	tick := time.NewTicker(1000 * time.Millisecond)
	defer tick.Stop()

	lastD, lastU, _ := getTotalBytes()

	for range tick.C {
		currD, currU, err := getTotalBytes()
		if err == nil {
			n.d <- float64(currD - lastD)
			n.u <- float64(currU - lastU)
			lastD, lastU = currD, currU
		}
	}
}

// 执行 Ping 采样的核心工具
func fetchPing(addr string, count int) PingResult {
	pinger, err := ping.NewPinger(addr)
	if err != nil {
		return PingResult{}
	}

	// 特权模式在 Windows 下必须为 true，Linux 下取决于系统配置
	pinger.SetPrivileged(true)
	pinger.Count = count
	pinger.Timeout = time.Millisecond * 3000 // 给予充足的超时时间

	err = pinger.Run()
	if err != nil {
		return PingResult{}
	}

	stats := pinger.Statistics()
	return PingResult{
		Latency: stats.AvgRtt,
		Loss:    stats.PacketLoss,
	}
}

// 网络质量监控协程
func (n *NetInfo) networkMonitor() {
	// 自动发现网关
	gwIP, err := gateway.DiscoverGateway()
	gwAddr := "192.168.1.1"
	if err == nil {
		gwAddr = gwIP.String()
	}

	pubAddr := "223.5.5.5" // 阿里云 DNS

	// Ping 采样频率：每 2 秒一次
	ticker := time.NewTicker(2000 * time.Millisecond)
	defer ticker.Stop()

	for range ticker.C {
		// 并行执行网关和公网 Ping，互不阻塞
		go func() {
			n.gwResult <- fetchPing(gwAddr, 3)
		}()

		go func() {
			n.pubResult <- fetchPing(pubAddr, 3)
		}()
	}
}

func main() {
	// 初始化通道
	info := NetInfo{
		d:         make(chan float64),
		u:         make(chan float64),
		gwResult:  make(chan PingResult),
		pubResult: make(chan PingResult),
	}

	// 启动后台协程
	go info.trafficController()
	go info.networkMonitor()

	// 缓存最新数据用于展示
	var (
		curD, curU float64
		curGW      PingResult
		curPub     PingResult
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
		case curD = <-info.d:
		case curU = <-info.u:
		case curGW = <-info.gwResult:
		case curPub = <-info.pubResult:
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
