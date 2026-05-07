package logger

import (
	"encoding/json"
	"fmt"
	"go-net-log/internal/fetcher"
	"os"
	"sync"
	"time"
)

// LogEntry 仅在包内使用，用于 JSON 序列化
type LogEntry struct {
	Timestamp  string  `json:"ts"`
	DownKB     float64 `json:"down_kb"`
	UpKB       float64 `json:"up_kb"`
	GWLatency  float64 `json:"gw_lat_ms"`
	GWLoss     float64 `json:"gw_loss"`
	PubLatency float64 `json:"pub_lat_ms"`
	PubLoss    float64 `json:"pub_loss"`
}

// Logger 结构体，持有文件句柄
type Logger struct {
	mu   sync.Mutex
	file *os.File
}

func nextMidnight() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
}

// OpenLogger 初始化日志文件，并启动零点切换协程
func OpenLogger() *Logger {
	l := &Logger{}
	l.openFile()

	go l.rotateAtMidnight()

	return l
}

func (l *Logger) openFile() {
	now := time.Now()
	filename := fmt.Sprintf("network_diag_%s.jsonl", now.Format("2006-01-02"))
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}
	l.file = f
}

func (l *Logger) rotateAtMidnight() {
	timer := time.NewTimer(time.Until(nextMidnight()))
	defer timer.Stop()

	for {
		<-timer.C

		l.mu.Lock()
		l.file.Close()
		l.openFile()
		l.mu.Unlock()

		timer.Reset(time.Until(nextMidnight()))
	}
}

// Close 关闭文件
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file.Close()
}

// Record 记录一次快照数据
func (l *Logger) Record(d, u float64, gw, pub fetcher.PingResult) error {
	entry := LogEntry{
		Timestamp:  time.Now().Format(time.RFC3339Nano),
		DownKB:     d / 1024,
		UpKB:       u / 1024,
		GWLatency:  float64(gw.Latency.Microseconds()) / 1000.0,
		GWLoss:     gw.Loss,
		PubLatency: float64(pub.Latency.Microseconds()) / 1000.0,
		PubLoss:    pub.Loss,
	}

	jsonData, err := json.Marshal(entry)
	if err != nil {
		return err
	}

	l.mu.Lock()
	defer l.mu.Unlock()
	_, err = l.file.Write(append(jsonData, '\n'))
	return err
}
