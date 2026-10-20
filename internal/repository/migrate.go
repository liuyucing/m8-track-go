package repository

import (
	"context"
	"database/sql"
	"log"
)

// localSchemaStmts 幂等的本地表结构补列语句。每条用 COL_LENGTH 守护，可重复执行。
var localSchemaStmts = []string{
	`IF COL_LENGTH('dbo.track_sync_record','ship_date') IS NULL ALTER TABLE dbo.track_sync_record ADD ship_date DATETIME NULL;`,
	`IF COL_LENGTH('dbo.track_sync_record','delivered_date') IS NULL ALTER TABLE dbo.track_sync_record ADD delivered_date DATETIME NULL;`,
	`IF COL_LENGTH('dbo.track_sync_detail','stage') IS NULL ALTER TABLE dbo.track_sync_detail ADD stage NVARCHAR(64) NULL;`,
	`IF COL_LENGTH('dbo.track_sync_detail','location') IS NULL ALTER TABLE dbo.track_sync_detail ADD location NVARCHAR(255) NULL;`,
}

// EnsureLocalSchema 幂等地为本地同步表补齐所需列。失败仅记录日志、不中断启动。
func EnsureLocalSchema(ctx context.Context, db *sql.DB) {
	for _, stmt := range localSchemaStmts {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			log.Printf("本地表结构迁移跳过（列可能已存在或无权限）: %v", err)
		}
	}
}
