package logger

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const maxAge = 7 // 保留最近 7 天的日志

var logFile *os.File

// Init 初始化文件日志，logDir 为日志文件所在目录
// 日志文件命名格式: m8-track-{date}.log
func Init(logDir string) error {
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("创建日志目录失败: %w", err)
	}

	cleanOldLogs(logDir)

	filename := fmt.Sprintf("m8-track-%s.log", time.Now().Format("2006-01-02"))
	filePath := filepath.Join(logDir, filename)

	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("打开日志文件失败: %w", err)
	}

	logFile = f

	// 同时输出到 stdout 和文件
	log.SetOutput(io.MultiWriter(os.Stdout, logFile))
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	return nil
}

// Close 关闭日志文件
func Close() {
	if logFile != nil {
		logFile.Close()
		logFile = nil
	}
}

// cleanOldLogs 删除超过 maxAge 天的日志文件
func cleanOldLogs(logDir string) {
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return
	}

	cutoff := time.Now().AddDate(0, 0, -maxAge)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasPrefix(name, "m8-track-") || !strings.HasSuffix(name, ".log") {
			continue
		}
		// 解析文件名中的日期: m8-track-2026-06-11.log
		dateStr := strings.TrimPrefix(name, "m8-track-")
		dateStr = strings.TrimSuffix(dateStr, ".log")
		fileDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}
		if fileDate.Before(cutoff) {
			os.Remove(filepath.Join(logDir, name))
		}
	}
}
