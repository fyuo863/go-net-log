package net

// 数据获取
import (
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
	D         chan float64    // 下载速率 (Bytes/s)
	U         chan float64    // 上传速率 (Bytes/s)
	GwResult  chan PingResult // 网关质量数据
	PubResult chan PingResult // 公网质量数据
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
func (n *NetInfo) TrafficController() {
	tick := time.NewTicker(1000 * time.Millisecond)
	defer tick.Stop()

	lastD, lastU, _ := getTotalBytes()

	for range tick.C {
		currD, currU, err := getTotalBytes()
		if err == nil {
			n.D <- float64(currD - lastD)
			n.U <- float64(currU - lastU)
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
func (n *NetInfo) NetworkMonitor() {
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
			n.GwResult <- fetchPing(gwAddr, 3)
		}()

		go func() {
			n.PubResult <- fetchPing(pubAddr, 3)
		}()
	}
}
