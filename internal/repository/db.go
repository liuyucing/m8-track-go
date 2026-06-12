package repository

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	"m8-track-go/config"

	_ "github.com/microsoft/go-mssqldb"
)

// MustOpenDB 打开 SQL Server 连接池，失败则 panic
func MustOpenDB(cfg config.DatabaseConfig) *sql.DB {
	dsn := cfg.DSN()
	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		panic(fmt.Sprintf("打开数据库连接失败: %v", err))
	}

	db.SetMaxOpenConns(cfg.MaxOpenConns)
	db.SetMaxIdleConns(cfg.MaxIdleConns)
	db.SetConnMaxLifetime(30 * time.Minute)
	// 空闲连接超过该时长即关闭，避免把被网络中间设备/数据库静默丢弃的“僵尸连接”
	// 再次交给查询使用而无限阻塞。
	db.SetConnMaxIdleTime(5 * time.Minute)

	// 验证连接
	if err := db.Ping(); err != nil {
		panic(fmt.Sprintf("数据库连接测试失败: %v", err))
	}

	log.Println("数据库连接成功")
	return db
}
