package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config 顶层配置结构
type Config struct {
	Database  DatabaseConfig  `yaml:"database"`
	Track17   Track17Config   `yaml:"track17"`
	Scheduler SchedulerConfig `yaml:"scheduler"`
	Query     QueryConfig     `yaml:"query"`
	App       AppConfig       `yaml:"app"`
}

// DatabaseConfig 数据库连接配置
type DatabaseConfig struct {
	Host         string `yaml:"host"`
	Port         int    `yaml:"port"`
	Name         string `yaml:"name"`
	Username     string `yaml:"username"`
	Password     string `yaml:"password"`
	Encrypt      bool   `yaml:"encrypt"`
	TrustCert    bool   `yaml:"trust_cert"`
	MaxOpenConns int    `yaml:"max_open_conns"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
}

// DSN 构建 Go mssqldb 驱动的连接字符串
func (d *DatabaseConfig) DSN() string {
	encrypt := "false"
	if d.Encrypt {
		encrypt = "true"
	}
	trust := "false"
	if d.TrustCert {
		trust = "true"
	}
	return fmt.Sprintf("sqlserver://%s:%s@%s:%d?database=%s&encrypt=%s&trustservercertificate=%s",
		d.Username, d.Password, d.Host, d.Port, d.Name, encrypt, trust)
}

// Track17Config 17track API 配置
type Track17Config struct {
	APIKey        string `yaml:"api_key"`
	BaseURL       string `yaml:"base_url"`
	BatchSize     int    `yaml:"batch_size"`
	HTTPTimeoutMs int    `yaml:"http_timeout_ms"`
}

// SchedulerConfig 定时任务配置
type SchedulerConfig struct {
	Cron               string `yaml:"cron"`
	Enabled            bool   `yaml:"enabled"`
	SyncTimeoutSeconds int    `yaml:"sync_timeout_seconds"`
}

// QueryConfig 查询条件配置
type QueryConfig struct {
	OrderDateFilter string `yaml:"order_date_filter"`
}

// AppConfig 应用级配置
type AppConfig struct {
	LogLevel string `yaml:"log_level"`
}

// MustLoad 加载配置文件，失败则 panic
func MustLoad(path string) *Config {
	cfg, err := Load(path)
	if err != nil {
		panic(err)
	}
	return cfg
}

// Load 加载配置文件
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件失败 %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	applyDefaults(&cfg)
	return &cfg, nil
}

// DefaultConfig 返回带默认值的空配置
func DefaultConfig() *Config {
	cfg := &Config{
		Database: DatabaseConfig{
			Port:         3366,
			Name:         "FumaCRM8",
			Username:     "sa",
			Encrypt:      false,
			TrustCert:    true,
			MaxOpenConns: 10,
			MaxIdleConns: 5,
		},
		Track17: Track17Config{
			BaseURL:       "https://api.17track.net/track/v2.4",
			BatchSize:     40,
			HTTPTimeoutMs: 30000,
		},
		Scheduler: SchedulerConfig{
			Cron:               "0 0 3,9,15,21 * * *",
			Enabled:            true,
			SyncTimeoutSeconds: 900,
		},
		Query: QueryConfig{
			OrderDateFilter: "2026-05-01",
		},
		App: AppConfig{
			LogLevel: "debug",
		},
	}
	return cfg
}

// IsConfigured 检查必要配置是否已填写
func (c *Config) IsConfigured() bool {
	return c.Database.Host != "" &&
		c.Database.Host != "你的数据库地址" &&
		c.Database.Password != "" &&
		c.Database.Password != "你的密码" &&
		c.Track17.APIKey != "" &&
		c.Track17.APIKey != "你的17track API密钥"
}

// Save 保存配置到文件
func (c *Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

func applyDefaults(cfg *Config) {
	if cfg.Track17.BatchSize == 0 {
		cfg.Track17.BatchSize = 40
	}
	if cfg.Track17.HTTPTimeoutMs == 0 {
		cfg.Track17.HTTPTimeoutMs = 30000
	}
	if cfg.Database.MaxOpenConns == 0 {
		cfg.Database.MaxOpenConns = 10
	}
	if cfg.Database.MaxIdleConns == 0 {
		cfg.Database.MaxIdleConns = 5
	}
	if cfg.Scheduler.SyncTimeoutSeconds == 0 {
		cfg.Scheduler.SyncTimeoutSeconds = 900
	}
}
