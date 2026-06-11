package main

import (
	"database/sql"
	"embed"
	"log"
	"os"
	"path/filepath"

	"m8-track-go/config"
	"m8-track-go/internal/app"
	"m8-track-go/internal/logger"
	"m8-track-go/internal/repository"
	"m8-track-go/internal/service"
	"m8-track-go/internal/trackapi"

	"github.com/wailsapp/wails/v3/pkg/application"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	// 初始化文件日志：日志目录为 exe 所在目录的 logs/ 子目录
	exePath, _ := os.Executable()
	logDir := filepath.Join(filepath.Dir(exePath), "logs")
	if err := logger.Init(logDir); err != nil {
		log.Printf("初始化文件日志失败: %v", err)
	}
	defer logger.Close()

	// 加载配置：如果文件不存在或内容不完整，使用默认配置
	configPath := "config.yaml"
	var cfg *config.Config
	if cfgData, err := config.Load(configPath); err != nil {
		log.Printf("配置加载失败，使用默认配置: %v", err)
		cfg = config.DefaultConfig()
	} else {
		cfg = cfgData
	}

	// 根据配置状态决定是否初始化数据库等依赖
	var (
		db            *sql.DB
		shipOrderRepo *repository.ShipOrderRepo
		recordRepo    *repository.TrackRecordRepo
		detailRepo    *repository.TrackDetailRepo
		syncService   *service.TrackSyncService
		scheduler     *service.Scheduler
	)

	if cfg.IsConfigured() {
		db = repository.MustOpenDB(cfg.Database)
		defer db.Close()

		shipOrderRepo = repository.NewShipOrderRepo(db, cfg.Query)
		recordRepo = repository.NewTrackRecordRepo(db)
		detailRepo = repository.NewTrackDetailRepo(db)
		trackClient := trackapi.NewClient(cfg.Track17)

		syncService = service.NewTrackSync(
			shipOrderRepo, recordRepo, detailRepo,
			trackClient, cfg.Track17.BatchSize,
		)
		scheduler = service.NewScheduler(cfg.Scheduler, syncService)
	} else {
		log.Println("配置未完成，将在GUI中进行配置")
	}

	appService := app.NewAppService(
		cfg, configPath, scheduler, syncService,
		recordRepo, detailRepo, shipOrderRepo,
	)

	// 创建 Wails 应用
	wailsApp := application.New(application.Options{
		Name:        "M8 物流轨迹同步",
		Description: "M8物流轨迹同步服务",
		Services: []application.Service{
			application.NewService(appService),
		},
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(assets),
		},
	})

	// 创建主窗口
	wailsApp.Window.NewWithOptions(application.WebviewWindowOptions{
		Title:  "M8 物流轨迹同步",
		Width:  1200,
		Height: 800,
	})

	if err := wailsApp.Run(); err != nil {
		log.Fatal(err)
	}
}
