package service

import (
	"context"
	"log"
	"sync"
	"time"

	"m8-track-go/config"

	"github.com/robfig/cron/v3"
)

// Scheduler 定时调度器，替代 Java 的 @Scheduled
type Scheduler struct {
	cron        *cron.Cron
	syncService *TrackSyncService
	enabled     bool
	spec        string
	mu          sync.Mutex
	lastRunTime *time.Time
	lastRunErr  error
	isRunning   bool
}

// NewScheduler 创建调度器
func NewScheduler(cfg config.SchedulerConfig, syncService *TrackSyncService) *Scheduler {
	return &Scheduler{
		cron:        cron.New(cron.WithSeconds()), // 支持 6 字段 cron 格式
		syncService: syncService,
		enabled:     cfg.Enabled,
		spec:        cfg.Cron,
	}
}

// Start 启动定时任务
func (s *Scheduler) Start() error {
	if !s.enabled {
		log.Println("调度器已禁用")
		return nil
	}

	_, err := s.cron.AddFunc(s.spec, s.runSync)
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

// runSync 执行同步（防重入）
func (s *Scheduler) runSync() {
	s.mu.Lock()
	if s.isRunning {
		s.mu.Unlock()
		log.Println("上一次同步尚未完成，跳过本次")
		return
	}
	s.isRunning = true
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.isRunning = false
		now := time.Now()
		s.lastRunTime = &now
		s.mu.Unlock()
	}()

	log.Println("开始定时同步...")
	ctx := context.Background()
	if err := s.syncService.SyncAll(ctx); err != nil {
		s.mu.Lock()
		s.lastRunErr = err
		s.mu.Unlock()
		log.Printf("定时同步失败: %v", err)
	} else {
		s.mu.Lock()
		s.lastRunErr = nil
		s.mu.Unlock()
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
