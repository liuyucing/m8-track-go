package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"m8-track-go/internal/model"
)

// TrackDetailRepo track_sync_detail 表 CRUD
type TrackDetailRepo struct {
	db *sql.DB
}

// NewTrackDetailRepo 创建 TrackDetailRepo
func NewTrackDetailRepo(db *sql.DB) *TrackDetailRepo {
	return &TrackDetailRepo{db: db}
}

// Insert 插入一条轨迹详情
func (r *TrackDetailRepo) Insert(ctx context.Context, detail *model.TrackSyncDetail) error {
	detail.MDNo = strings.TrimSpace(detail.MDNo)

	query := `INSERT INTO track_sync_detail (md_no, track_status, event_desc, event_time, create_time)
              VALUES (@p1, @p2, @p3, @p4, @p5);
              SELECT SCOPE_IDENTITY();`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		detail.MDNo, detail.TrackStatus, detail.EventDesc,
		detail.EventTime, detail.CreateTime,
	).Scan(&id)
	if err != nil {
		return fmt.Errorf("插入轨迹详情失败: %w", err)
	}
	detail.ID = id
	return nil
}

// ListByMDNo 查询某运单号的轨迹事件
func (r *TrackDetailRepo) ListByMDNo(ctx context.Context, mdNo string, limit int) ([]model.TrackSyncDetail, error) {
	mdNo = strings.TrimSpace(mdNo)

	query := `SELECT TOP (@p1) id, md_no, track_status, event_desc, event_time, create_time
              FROM track_sync_detail WHERE md_no = @p2 ORDER BY create_time DESC`

	rows, err := r.db.QueryContext(ctx, query, limit, mdNo)
	if err != nil {
		return nil, fmt.Errorf("查询轨迹详情失败 mdNo=%s: %w", mdNo, err)
	}
	defer rows.Close()

	var details []model.TrackSyncDetail
	for rows.Next() {
		var d model.TrackSyncDetail
		if err := rows.Scan(&d.ID, &d.MDNo, &d.TrackStatus, &d.EventDesc, &d.EventTime, &d.CreateTime); err != nil {
			return nil, fmt.Errorf("扫描轨迹详情失败: %w", err)
		}
		details = append(details, d)
	}
	return details, rows.Err()
}

// ListRecent 查询最近的轨迹事件
func (r *TrackDetailRepo) ListRecent(ctx context.Context, limit int) ([]model.TrackSyncDetail, error) {
	query := `SELECT TOP (@p1) id, md_no, track_status, event_desc, event_time, create_time
              FROM track_sync_detail ORDER BY create_time DESC`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("查询最近轨迹详情失败: %w", err)
	}
	defer rows.Close()

	var details []model.TrackSyncDetail
	for rows.Next() {
		var d model.TrackSyncDetail
		if err := rows.Scan(&d.ID, &d.MDNo, &d.TrackStatus, &d.EventDesc, &d.EventTime, &d.CreateTime); err != nil {
			return nil, fmt.Errorf("扫描轨迹详情失败: %w", err)
		}
		details = append(details, d)
	}
	return details, rows.Err()
}
