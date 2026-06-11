package app

import (
	"context"
	"fmt"
	"log"
	"strings"

	"m8-track-go/config"
	"m8-track-go/internal/model"
	"m8-track-go/internal/repository"
	"m8-track-go/internal/service"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// AppService Wails 暴露给前端的唯一服务
type AppService struct {
	cfg           *config.Config
	configPath    string
	scheduler     *service.Scheduler
	syncService   *service.TrackSyncService
	recordRepo    *repository.TrackRecordRepo
	detailRepo    *repository.TrackDetailRepo
	shipOrderRepo *repository.ShipOrderRepo
}

// NewAppService 创建 AppService
func NewAppService(
	cfg *config.Config,
	configPath string,
	scheduler *service.Scheduler,
	syncService *service.TrackSyncService,
	recordRepo *repository.TrackRecordRepo,
	detailRepo *repository.TrackDetailRepo,
	shipOrderRepo *repository.ShipOrderRepo,
) *AppService {
	return &AppService{
		cfg:           cfg,
		configPath:    configPath,
		scheduler:     scheduler,
		syncService:   syncService,
		recordRepo:    recordRepo,
		detailRepo:    detailRepo,
		shipOrderRepo: shipOrderRepo,
	}
}

// OnStartup Wails 生命周期 - 服务启动时调用
func (s *AppService) OnStartup(ctx context.Context, options application.ServiceOptions) error {
	log.Println("AppService 启动")
	if s.cfg.IsConfigured() && s.scheduler != nil {
		if err := s.scheduler.Start(); err != nil {
			log.Printf("启动调度器失败: %v", err)
		}
	} else {
		log.Println("配置未完成，请先在配置页面填写数据库和API信息")
	}
	return nil
}

// OnShutdown Wails 生命周期 - 服务关闭时调用
func (s *AppService) OnShutdown() error {
	log.Println("AppService 关闭")
	if s.scheduler != nil {
		s.scheduler.Stop()
	}
	return nil
}

// IsConfigured 检查是否已配置完成
func (s *AppService) IsConfigured() bool {
	return s.cfg.IsConfigured()
}

// SaveConfig 保存配置（从前端传入完整配置）
func (s *AppService) SaveConfig(cfgData map[string]interface{}) error {
	// 更新数据库配置
	if db, ok := cfgData["database"].(map[string]interface{}); ok {
		if v, ok := db["host"].(string); ok {
			s.cfg.Database.Host = v
		}
		if v, ok := db["port"].(float64); ok {
			s.cfg.Database.Port = int(v)
		}
		if v, ok := db["name"].(string); ok {
			s.cfg.Database.Name = v
		}
		if v, ok := db["username"].(string); ok {
			s.cfg.Database.Username = v
		}
		if v, ok := db["password"].(string); ok && !isMasked(v) {
			s.cfg.Database.Password = v
		}
	}
	// 更新17track配置
	if t, ok := cfgData["track17"].(map[string]interface{}); ok {
		if v, ok := t["api_key"].(string); ok && !isMasked(v) {
			s.cfg.Track17.APIKey = v
		}
		if v, ok := t["base_url"].(string); ok {
			s.cfg.Track17.BaseURL = v
		}
		if v, ok := t["batch_size"].(float64); ok {
			s.cfg.Track17.BatchSize = int(v)
		}
	}
	// 更新调度器配置
	if sc, ok := cfgData["scheduler"].(map[string]interface{}); ok {
		if v, ok := sc["cron"].(string); ok {
			s.cfg.Scheduler.Cron = v
		}
		if v, ok := sc["enabled"].(bool); ok {
			s.cfg.Scheduler.Enabled = v
		}
	}
	// 更新查询配置
	if q, ok := cfgData["query"].(map[string]interface{}); ok {
		if v, ok := q["order_date_filter"].(string); ok {
			s.cfg.Query.OrderDateFilter = v
		}
	}

	if err := s.cfg.Save(s.configPath); err != nil {
		return fmt.Errorf("保存配置文件失败: %w", err)
	}
	log.Println("配置已保存，请重启应用以使配置生效")
	return nil
}

// --- Dashboard ---

// DashboardStats 仪表盘统计数据
type DashboardStats struct {
	TotalOrders      int64  `json:"totalOrders"`
	Delivered        int64  `json:"delivered"`
	InTransit        int64  `json:"inTransit"`
	IsSyncRunning    bool   `json:"isSyncRunning"`
	LastSyncTime     string `json:"lastSyncTime"`
	LastSyncError    string `json:"lastSyncError"`
	SchedulerEnabled bool   `json:"schedulerEnabled"`
	CronSpec         string `json:"cronSpec"`
}

// GetDashboardStats 获取仪表盘统计数据
func (s *AppService) GetDashboardStats() DashboardStats {
	if s.recordRepo == nil {
		return DashboardStats{SchedulerEnabled: s.cfg.Scheduler.Enabled, CronSpec: s.cfg.Scheduler.Cron}
	}
	ctx := context.Background()

	total, _ := s.recordRepo.CountAll(ctx)
	delivered, _ := s.recordRepo.CountByStatus(ctx, true)
	inTransit := total - delivered

	status := s.scheduler.GetStatus()

	stats := DashboardStats{
		TotalOrders:      total,
		Delivered:        delivered,
		InTransit:        inTransit,
		IsSyncRunning:    status.IsRunning,
		SchedulerEnabled: status.Enabled,
		CronSpec:         status.CronSpec,
	}

	if status.LastRunTime != nil {
		stats.LastSyncTime = status.LastRunTime.Format("2006-01-02 15:04:05")
	}
	if status.LastRunError != nil {
		stats.LastSyncError = *status.LastRunError
	}

	return stats
}

// --- Manual Sync ---

// TriggerSync 手动触发同步
func (s *AppService) TriggerSync() error {
	if s.scheduler == nil || s.syncService == nil {
		return fmt.Errorf("请先完成配置并重启应用")
	}
	status := s.scheduler.GetStatus()
	if status.IsRunning {
		return fmt.Errorf("同步正在进行中，请稍后")
	}

	go func() {
		ctx := context.Background()
		if err := s.syncService.SyncAll(ctx); err != nil {
			log.Printf("手动同步失败: %v", err)
		}
	}()

	return nil
}

// --- Orders ---

// OrderListItem 订单列表项
type OrderListItem struct {
	MDNo         string  `json:"mdNo"`
	FID          string  `json:"fid"`
	TrackStatus  *string `json:"trackStatus"`
	LastEvent    *string `json:"lastEvent"`
	IsDelivered  bool    `json:"isDelivered"`
	LastSyncTime string  `json:"lastSyncTime"`
}

// GetOrders 获取订单列表
func (s *AppService) GetOrders(pageSize int) []OrderListItem {
	if s.recordRepo == nil {
		return nil
	}
	ctx := context.Background()
	records, err := s.recordRepo.ListRecent(ctx, pageSize)
	if err != nil {
		log.Printf("获取订单列表失败: %v", err)
		return nil
	}

	items := make([]OrderListItem, 0, len(records))
	for _, r := range records {
		item := OrderListItem{
			MDNo:        r.MDNo,
			FID:         r.FID,
			TrackStatus: r.TrackStatus,
			LastEvent:   r.LastEvent,
			IsDelivered: r.IsDelivered,
		}
		if r.LastSyncTime != nil {
			item.LastSyncTime = r.LastSyncTime.Format("2006-01-02 15:04:05")
		}
		items = append(items, item)
	}
	return items
}

// GetOrderDetails 获取单个订单的轨迹事件
func (s *AppService) GetOrderDetails(mdNo string) []model.TrackSyncDetail {
	if s.detailRepo == nil {
		return nil
	}
	ctx := context.Background()
	details, err := s.detailRepo.ListByMDNo(ctx, mdNo, 50)
	if err != nil {
		log.Printf("获取订单详情失败 mdNo=%s: %v", mdNo, err)
		return nil
	}
	return details
}

// --- Logs ---

// LogEntry 日志条目
type LogEntry struct {
	ID         int64  `json:"id"`
	MDNo       string `json:"mdNo"`
	Status     string `json:"status"`
	EventDesc  string `json:"eventDesc"`
	EventTime  string `json:"eventTime"`
	CreateTime string `json:"createTime"`
}

// GetRecentLogs 获取最近的同步日志
func (s *AppService) GetRecentLogs(limit int) []LogEntry {
	if s.detailRepo == nil {
		return nil
	}
	ctx := context.Background()
	details, err := s.detailRepo.ListRecent(ctx, limit)
	if err != nil {
		log.Printf("获取日志失败: %v", err)
		return nil
	}

	entries := make([]LogEntry, 0, len(details))
	for _, d := range details {
		entry := LogEntry{
			ID:         d.ID,
			MDNo:       d.MDNo,
			CreateTime: d.CreateTime.Format("2006-01-02 15:04:05"),
		}
		if d.TrackStatus != nil {
			entry.Status = *d.TrackStatus
		}
		if d.EventDesc != nil {
			entry.EventDesc = *d.EventDesc
		}
		if d.EventTime != nil {
			entry.EventTime = d.EventTime.Format("2006-01-02 15:04:05")
		}
		entries = append(entries, entry)
	}
	return entries
}

// --- Config ---

// GetConfig 获取当前配置（密码脱敏）
func (s *AppService) GetConfig() map[string]interface{} {
	cfg := s.cfg
	masked := maskSecret(cfg.Database.Password)
	maskedKey := maskSecret(cfg.Track17.APIKey)

	return map[string]interface{}{
		"database": map[string]interface{}{
			"host":     cfg.Database.Host,
			"port":     cfg.Database.Port,
			"name":     cfg.Database.Name,
			"username": cfg.Database.Username,
			"password": masked,
		},
		"track17": map[string]interface{}{
			"base_url":   cfg.Track17.BaseURL,
			"batch_size": cfg.Track17.BatchSize,
			"api_key":    maskedKey,
		},
		"scheduler": map[string]interface{}{
			"cron":    cfg.Scheduler.Cron,
			"enabled": cfg.Scheduler.Enabled,
		},
		"query": map[string]interface{}{
			"order_date_filter": cfg.Query.OrderDateFilter,
		},
	}
}

func maskSecret(s string) string {
	if len(s) <= 4 {
		return "****"
	}
	return s[:2] + strings.Repeat("*", len(s)-4) + s[len(s)-2:]
}

// isMasked 判断是否是 maskSecret 生成的脱敏值，避免将脱敏值回写到配置文件
func isMasked(s string) bool {
	if len(s) <= 4 {
		return s == "****"
	}
	mid := s[2 : len(s)-2]
	if len(mid) == 0 {
		return false
	}
	for _, c := range mid {
		if c != '*' {
			return false
		}
	}
	return true
}
