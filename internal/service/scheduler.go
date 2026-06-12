package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"runtime/debug"
	"sync"
	"time"

	"m8-track-go/config"

	"encoding/json"
	"os"
	"path/filepath"

	"github.com/robfig/cron/v3"
)

// syncState 持久化状态（写入 sync_state.json）
type syncState struct {
	LastRunTime  string `json:"lastRunTime,omitempty"`
	LastRunError string `json:"lastRunError,omitempty"`
}

// Scheduler 定时调度器，替代 Java 的 @Scheduled
type Scheduler struct {
	cron        *cron.Cron
	syncService *TrackSyncService
	enabled     bool
	spec        string
	stateFile   string
	mu          sync.Mutex
	lastRunTime *time.Time
	lastRunErr  error
	isRunning   bool
	syncTimeout time.Duration
}

// NewScheduler 创建调度器，stateDir 用于存放 sync_state.json
func NewScheduler(cfg config.SchedulerConfig, syncService *TrackSyncService, stateDir string) *Scheduler {
	syncTimeout := time.Duration(cfg.SyncTimeoutSeconds) * time.Second
	if syncTimeout <= 0 {
		syncTimeout = 15 * time.Minute // 兜底，正常应由 config applyDefaults 提供
	}
	s := &Scheduler{
		cron:        cron.New(cron.WithSeconds()), // 支持 6 字段 cron 格式
		syncService: syncService,
		enabled:     cfg.Enabled,
		spec:        cfg.Cron,
		stateFile:   filepath.Join(stateDir, "sync_state.json"),
		syncTimeout: syncTimeout,
	}
	s.loadState()
	return s
}

// Start 启动定时任务
func (s *Scheduler) Start() error {
	if !s.enabled {
		log.Println("调度器已禁用")
		return nil
	}

	_, err := s.cron.AddFunc(s.spec, s.RunSync)
	if err != nil {
		return err
	}

	s.cron.Start()
	log.Printf("调度器已启动，cron: %s", s.spec)
	return nil
}

// Stop 停止定时任务
func (s *Scheduler) Stop() {
	if s.cron != nil {
		ctx := s.cron.Stop()
		<-ctx.Done()
		log.Println("调度器已停止")
	}
}

// RunSync 执行同步（防重入），可由调度器定时调用或手动触发
func (s *Scheduler) RunSync() {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		log.Println("上一次同步尚未完成，跳过本次")
		return
	}
	s.isRunning = true
	s.mu.Unlock()

	defer func() {
		// 兜底 recover：同步在 goroutine 中执行，任何 panic 若不恢复会导致整个进程崩溃（闪退）。
		// 捕获后把 panic 值与完整堆栈写入日志文件（panic 默认走 stderr，GUI 下不可见），
		// 并记为本次 lastRunErr（仪表盘可见）；随后正常复位 isRunning 并持久化状态。
		if r := recover(); r != nil {
			log.Printf("同步发生 panic（已恢复，未崩溃）: %v\n%s", r, debug.Stack())
			s.mu.Lock()
			s.lastRunErr = fmt.Errorf("同步 panic: %v", r)
			s.mu.Unlock()
		}
		s.mu.Lock()
		s.isRunning = false
		now := time.Now()
		s.lastRunTime = &now
		s.mu.Unlock()
		s.saveState()
	}()

	log.Println("开始定时同步...")
	// 为整次同步设置超时上限：即使某条 SQL Server 连接变成僵尸、查询无限阻塞，
	// 到点也会因 ctx 取消而返回，defer 随之执行、isRunning 复位，下次同步可正常触发。
	ctx, cancel := context.WithTimeout(context.Background(), s.syncTimeout)
	defer cancel()

	err := s.syncService.SyncAll(ctx)
	// SyncAll 内部部分错误会被吞掉（如 HTTP 批次失败仅 log+continue），
	// 这里用 ctx.Err() 兜底，确保超时被如实上报，不会误报“完成”。
	if err == nil {
		err = ctx.Err()
	}

	s.mu.Lock()
	s.lastRunErr = err
	s.mu.Unlock()

	switch {
	case errors.Is(err, context.DeadlineExceeded):
		log.Printf("定时同步超时（超过 %v），已强制结束；下次调度或手动触发将自动恢复", s.syncTimeout)
	case err != nil:
		log.Printf("定时同步失败: %v", err)
	default:
		log.Println("定时同步完成")
	}
}

// SchedulerStatus 调度器状态
type SchedulerStatus struct {
	IsRunning    bool       `json:"isRunning"`
	Enabled      bool       `json:"enabled"`
	CronSpec     string     `json:"cronSpec"`
	LastRunTime  *time.Time `json:"lastRunTime"`
	LastRunError *string    `json:"lastRunError"`
}

// GetStatus 获取当前调度器状态
func (s *Scheduler) GetStatus() SchedulerStatus {
	s.mu.Lock()
	defer s.mu.Unlock()

	status := SchedulerStatus{
		IsRunning:   s.isRunning,
		Enabled:     s.enabled,
		CronSpec:    s.spec,
		LastRunTime: s.lastRunTime,
	}
	if s.lastRunErr != nil {
		errMsg := s.lastRunErr.Error()
		status.LastRunError = &errMsg
	}
	return status
}

// loadState 从文件恢复上次同步时间，重启后仪表盘仍能显示
func (s *Scheduler) loadState() {
	if s.stateFile == "" {
		return
	}
	data, err := os.ReadFile(s.stateFile)
	if err != nil {
		return // 文件不存在则忽略
	}
	var state syncState
	if err := json.Unmarshal(data, &state); err != nil {
		return
	}
	if state.LastRunTime != "" {
		t, err := time.Parse(time.RFC3339, state.LastRunTime)
		if err == nil {
			s.lastRunTime = &t
		}
	}
	if state.LastRunError != "" {
		s.lastRunErr = fmt.Errorf("%s", state.LastRunError)
	}
	log.Printf("恢复同步状态: lastRunTime=%v", s.lastRunTime)
}

// saveState 将上次同步时间写入文件
func (s *Scheduler) saveState() {
	if s.stateFile == "" {
		return
	}
	s.mu.Lock()
	state := syncState{}
	if s.lastRunTime != nil {
		state.LastRunTime = s.lastRunTime.Format(time.RFC3339)
	}
	if s.lastRunErr != nil {
		state.LastRunError = s.lastRunErr.Error()
	}
	s.mu.Unlock()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(s.stateFile), 0755)
	_ = os.WriteFile(s.stateFile, data, 0644)
}
