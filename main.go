package main

import (
	"fmt"
	"time"

	"github.com/go-ping/ping"
	"github.com/jackpal/gateway"
	"github.com/shirou/gopsutil/v4/net"
)

// NetInfo 结构体定义
type NetInfo struct {
	d          chan float64       // Download speed (or total)
	u          chan float64       // Upload speed (or total)
	gwLatency  chan time.Duration // 网关延迟
	pubLatency chan time.Duration // 公网延迟
	packetLoss chan float64       // 公网丢包率
}

func getTotalBytes() (uint64, uint64, error) {
	// 获取所有网卡的总计数
	counters, err := net.IOCounters(false) // false 表示返回所有网卡的聚合结果
	if err != nil {
		return 0, 0, err
	}
	// counters[0] 即为 "all" 的聚合数据
	return counters[0].BytesRecv, counters[0].BytesSent, nil
}

func (n *NetInfo) controller() {
	tick := time.NewTicker(1000 * time.Millisecond) // 每秒采样一次计算速率更准确
	defer tick.Stop()

	// 记录上一次的数据用于计算差值（速率）
	lastD, lastU, _ := getTotalBytes()

	for range tick.C {
		currD, currU, err := getTotalBytes()
		if err == nil {
			// 计算差值（每秒增加的字节数）
			deltaD := float64(currD - lastD)
			deltaU := float64(currU - lastU)

			// 发送到通道
			n.d <- deltaD
			n.u <- deltaU

			// 更新旧值
			lastD, lastU = currD, currU
		}

	}
}

func getPingStats(addr string, count int) (time.Duration, float64, error) {
	pinger, err := ping.NewPinger(addr)
	if err != nil {
		return 0, 0, err
	}

	// Windows 下需要设为 true，Linux 下如果不是 root 且没设 setcap 也需要
	pinger.SetPrivileged(true)
	pinger.Count = count
	pinger.Timeout = time.Millisecond * 2000 // 总超时

	err = pinger.Run() // 阻塞直到结束
	if err != nil {
		return 0, 0, err
	}

	stats := pinger.Statistics()
	return stats.AvgRtt, stats.PacketLoss, nil
}

func (n *NetInfo) networkMonitor() {
	// 获取网关 IP
	gwIP, err := gateway.DiscoverGateway()
	gwAddr := "192.168.1.1" // 默认备选
	if err == nil {
		gwAddr = gwIP.String()
	}

	pubAddr := "223.5.5.5" // 阿里云公共 DNS (国内推荐) 或 8.8.8.8

	ticker := time.NewTicker(2000 * time.Millisecond) // Ping 耗时较长，建议 2s 采样一次
	defer ticker.Stop()

	for range ticker.C {
		// 1. 获取网关延迟
		go func() {
			rtt, _, _ := getPingStats(gwAddr, 2)
			n.gwLatency <- rtt
		}()

		// 2. 获取公网延迟和丢包
		go func() {
			rtt, loss, _ := getPingStats(pubAddr, 3)
			n.pubLatency <- rtt
			n.packetLoss <- loss
		}()
	}
}

func main() {
	info := NetInfo{
		d:          make(chan float64),
		u:          make(chan float64),
		gwLatency:  make(chan time.Duration),
		pubLatency: make(chan time.Duration),
		packetLoss: make(chan float64),
	}

	// 启动流量监控
	go info.controller()
	// 启动延迟监控
	go info.networkMonitor()

	var d, u float64
	var gwL, pubL time.Duration
	var loss float64

	fmt.Println("正在启动高级网络诊断...")

	for {
		select {
		case d = <-info.d:
		case u = <-info.u:
		case gwL = <-info.gwLatency:
		case pubL = <-info.pubLatency:
		case loss = <-info.packetLoss:
		}

		// 格式化输出
		// \033[K 清除行末防止字符残留
		fmt.Printf("\r[流量] 下行:%7.2fKB/s 上行:%7.2fKB/s | [延迟] 网关:%-7v 公网:%-7v | [丢包]: %.1f%%\033[K",
			d/1024, u/1024, gwL, pubL, loss)
	}
}
