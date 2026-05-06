package main

import (
	"fmt"

	"time"

	"github.com/shirou/gopsutil/v4/net"
)

type NetInfo struct {
	d chan float64
	u chan float64
}

func getTotalBytes() (uint64, uint64, error) {
	// 获取所有网卡的总下载字节数和总上传字节数
	counters, err := net.IOCounters(true) // 获取所有网卡
	if err != nil {
		return 0, 0, err
	}
	var Utotal uint64
	var Dtotal uint64

	for _, c := range counters {
		Utotal += c.BytesRecv // 累计接收字节（下行）
		Dtotal += c.BytesSent // 累计发送字节（上行）
	}
	return Utotal, Dtotal, nil
}

func ticker(d chan float64, u chan float64) {
	start := time.Now()
	tick := time.NewTicker(500 * time.Millisecond)
	elapsed := func() time.Duration {
		return time.Since(start).Round(time.Millisecond)
	}
	for {
		select {
		case <-tick.C:
			Utotal, Dtotal, err := getTotalBytes()
			if err == nil {
				d <- float64(Utotal)
				u <- float64(Dtotal)
				//fmt.Printf("\r已运行: %s, 下载: %d 字节, 上传: %d 字节", elapsed(), Utotal, Dtotal)
			}
		default:
			fmt.Printf("\r已运行: %s", elapsed())
		}
	}
}

func (n *NetInfo) controller() {

	ticker(n.d, n.u)
	// Utotal, Dtotal, err := getTotalBytes()
	// if err == nil {
	// 	fmt.Println("开机以来的总下载字节数: ", Utotal)
	// 	fmt.Println("开机以来的总上传字节数: ", Dtotal)
	// }
}

func main() {
	info := NetInfo{
		d: make(chan float64),
		u: make(chan float64),
	}
	// d := make(chan float64) //下载
	// u := make(chan float64) //上传
	go info.controller()
	for {
		tempd := <-info.d
		tempu := <-info.u
		fmt.Printf("下载: %.2f 字节, 上传: %.2f 字节", tempd, tempu)
	}
}
