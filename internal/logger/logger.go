package logger

import (
	"encoding/json"
	"fmt"
	"go-net-log/internal/fetcher"
	"os"
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
	file *os.File
}

// NewLogger 初始化日志文件
func OpenLogger() *Logger {
	now := time.Now()
	filename := fmt.Sprintf("network_diag_%s.jsonl", now.Format("2006-01-02"))
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		panic(err)
	}

	return &Logger{file: f}
}

// Close 关闭文件
func (l *Logger) Close() error {
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

	_, err = l.file.Write(append(jsonData, '\n'))
	return err
}
