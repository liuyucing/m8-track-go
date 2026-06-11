package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"m8-track-go/internal/model"
)

// TrackRecordRepo track_sync_record 表 CRUD
type TrackRecordRepo struct {
	db *sql.DB
}

// NewTrackRecordRepo 创建 TrackRecordRepo
func NewTrackRecordRepo(db *sql.DB) *TrackRecordRepo {
	return &TrackRecordRepo{db: db}
}

// Insert 插入一条同步记录
func (r *TrackRecordRepo) Insert(ctx context.Context, record *model.TrackSyncRecord) error {
	record.MDNo = strings.TrimSpace(record.MDNo)

	query := `INSERT INTO track_sync_record (fid, md_no, track_status, last_event, last_event_time, last_sync_time, is_delivered, create_time, update_time)
              VALUES (@p1, @p2, @p3, @p4, @p5, @p6, @p7, @p8, @p9);
              SELECT SCOPE_IDENTITY();`

	var id int64
	err := r.db.QueryRowContext(ctx, query,
		record.FID, record.MDNo, record.TrackStatus, record.LastEvent,
		record.LastEventTime, record.LastSyncTime, record.IsDelivered,
		record.CreateTime, record.UpdateTime,
	).Scan(&id)
	if err != nil {
		return fmt.Errorf("插入同步记录失败: %w", err)
	}
	record.ID = id
	return nil
}

// GetByMDNos 根据运单号列表查询记录
func (r *TrackRecordRepo) GetByMDNos(ctx context.Context, mdNos []string) ([]model.TrackSyncRecord, error) {
	if len(mdNos) == 0 {
		return nil, nil
	}

	// 对每个 mdNo 做 trim
	trimmed := make([]string, len(mdNos))
	for i, mdNo := range mdNos {
		trimmed[i] = strings.TrimSpace(mdNo)
	}

	placeholders := make([]string, len(trimmed))
	args := make([]interface{}, len(trimmed))
	for i, mdNo := range trimmed {
		placeholders[i] = fmt.Sprintf("@p%d", i+1)
		args[i] = mdNo
	}

	query := fmt.Sprintf(`SELECT id, fid, md_no, track_status, last_event, last_event_time, last_sync_time, is_delivered, create_time, update_time
                          FROM track_sync_record WHERE md_no IN (%s)`, strings.Join(placeholders, ", "))

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("查询同步记录失败: %w", err)
	}
	defer rows.Close()

	var records []model.TrackSyncRecord
	for rows.Next() {
		var rec model.TrackSyncRecord
		if err := rows.Scan(&rec.ID, &rec.FID, &rec.MDNo, &rec.TrackStatus, &rec.LastEvent,
			&rec.LastEventTime, &rec.LastSyncTime, &rec.IsDelivered, &rec.CreateTime, &rec.UpdateTime); err != nil {
			return nil, fmt.Errorf("扫描同步记录失败: %w", err)
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}

// Update 更新同步记录
func (r *TrackRecordRepo) Update(ctx context.Context, record *model.TrackSyncRecord) error {
	query := `UPDATE track_sync_record SET
              track_status = @p1, last_event = @p2, last_event_time = @p3,
              last_sync_time = @p4, is_delivered = @p5, update_time = @p6
              WHERE id = @p7`

	_, err := r.db.ExecContext(ctx, query,
		record.TrackStatus, record.LastEvent, record.LastEventTime,
		record.LastSyncTime, record.IsDelivered, record.UpdateTime, record.ID)
	if err != nil {
		return fmt.Errorf("更新同步记录失败 id=%d: %w", record.ID, err)
	}
	return nil
}

// CountByStatus 按签收状态统计数量
func (r *TrackRecordRepo) CountByStatus(ctx context.Context, isDelivered bool) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM track_sync_record WHERE is_delivered = @p1", isDelivered).Scan(&count)
	return count, err
}

// CountAll 统计总记录数
func (r *TrackRecordRepo) CountAll(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM track_sync_record").Scan(&count)
	return count, err
}

// ListRecent 查询最近的同步记录
func (r *TrackRecordRepo) ListRecent(ctx context.Context, limit int) ([]model.TrackSyncRecord, error) {
	query := `SELECT TOP (@p1) id, fid, md_no, track_status, last_event, last_event_time, last_sync_time, is_delivered, create_time, update_time
              FROM track_sync_record ORDER BY update_time DESC`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("查询最近记录失败: %w", err)
	}
	defer rows.Close()

	var records []model.TrackSyncRecord
	for rows.Next() {
		var rec model.TrackSyncRecord
		if err := rows.Scan(&rec.ID, &rec.FID, &rec.MDNo, &rec.TrackStatus, &rec.LastEvent,
			&rec.LastEventTime, &rec.LastSyncTime, &rec.IsDelivered, &rec.CreateTime, &rec.UpdateTime); err != nil {
			return nil, fmt.Errorf("扫描记录失败: %w", err)
		}
		records = append(records, rec)
	}
	return records, rows.Err()
}
