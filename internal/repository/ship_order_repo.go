package repository

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	"m8-track-go/config"
	"m8-track-go/internal/model"
)

// ShipOrderRepo 物流订单数据访问，对应 Java ShipOrderMapper
type ShipOrderRepo struct {
	db         *sql.DB
	dateFilter string
}

// NewShipOrderRepo 创建 ShipOrderRepo
func NewShipOrderRepo(db *sql.DB, queryCfg config.QueryConfig) *ShipOrderRepo {
	return &ShipOrderRepo{db: db, dateFilter: queryCfg.OrderDateFilter}
}

// SelectPendingOrders 查询待处理运单：
//  1. 未签收的运单（TrackDelivered = 0）—— 常规出口，避免候选集无限增长；
//  2. 已签收但运单号被改过的运单（同一 scBNDtl.FID 在 track_sync_record 里记录的
//     md_no 与当前 scBNDtl.MDNo 不一致）—— 用于重新跟踪新号。
//
// 通过 LEFT JOIN track_sync_record 带出 OldMDNo（仅当检测到变更时非空）。
func (r *ShipOrderRepo) SelectPendingOrders(ctx context.Context) ([]model.ShipOrder, error) {
	query := `SELECT m.FID, d1.FID AS DtlFID, trim(d1.MDNo) AS MDNo, d1.FCkey, r.md_no AS OldMDNo
              FROM scbn m
              INNER JOIN scBNDtl d1 ON m.fid = d1.mfid
              LEFT JOIN track_sync_record r ON r.dtl_fid = d1.FID
              WHERE m.IsDeleted = 0
                AND m.CreateDate > @p1
                AND d1.MDNo IS NOT NULL
                AND d1.fckey IS NOT NULL
                AND (
                      d1.TrackDelivered = 0
                   OR (r.dtl_fid IS NOT NULL AND LTRIM(RTRIM(r.md_no)) <> LTRIM(RTRIM(d1.MDNo)))
                )`

	rows, err := r.db.QueryContext(ctx, query, r.dateFilter+" 00:00:00")
	if err != nil {
		return nil, fmt.Errorf("查询待处理运单失败: %w", err)
	}
	defer rows.Close()

	var orders []model.ShipOrder
	for rows.Next() {
		var o model.ShipOrder
		var oldMDNo sql.NullString
		if err := rows.Scan(&o.FID, &o.DtlFID, &o.MDNo, &o.FCKeY, &oldMDNo); err != nil {
			return nil, fmt.Errorf("扫描运单数据失败: %w", err)
		}
		o.MDNo = strings.TrimSpace(o.MDNo)
		if oldMDNo.Valid {
			old := strings.TrimSpace(oldMDNo.String)
			if old != "" && old != o.MDNo {
				o.OldMDNo = &old
			}
		}
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

// ResetDeliveredByFID 按 scBNDtl.FID（明细主键）将 TrackDelivered 重置为 0，
// 供运单号变更后重新跟踪使用。
func (r *ShipOrderRepo) ResetDeliveredByFID(ctx context.Context, dtlFID int64) error {
	result, err := r.db.ExecContext(ctx, "UPDATE scBNDtl SET TrackDelivered = 0 WHERE FID = @p1", dtlFID)
	if err != nil {
		return fmt.Errorf("重置签收状态失败 dtlFID=%d: %w", dtlFID, err)
	}
	if n, _ := result.RowsAffected(); n == 0 {
		log.Printf("重置签收状态未命中任何行 dtlFID=%d", dtlFID)
	}
	log.Printf("重置签收状态: dtlFID=%d", dtlFID)
	return nil
}

// UpdateFCtrackByFID 按 scBNDtl.FID（明细主键）更新 FCtrack。
// 用稳定主键而非可变的 MDNo 定位行，避免运单号变更后回写错行。
func (r *ShipOrderRepo) UpdateFCtrackByFID(ctx context.Context, dtlFID int64, value string) error {
	_, err := r.db.ExecContext(ctx, "UPDATE scBNDtl SET FCtrack = @p1 WHERE FID = @p2", value, dtlFID)
	if err != nil {
		return fmt.Errorf("更新 FCtrack 失败 dtlFID=%d: %w", dtlFID, err)
	}
	log.Printf("更新 FCtrack: dtlFID=%d, value=%s", dtlFID, value)
	return nil
}

// MarkDeliveredByFID 按 scBNDtl.FID（明细主键）标记为已签收。
func (r *ShipOrderRepo) MarkDeliveredByFID(ctx context.Context, dtlFID int64) error {
	_, err := r.db.ExecContext(ctx, "UPDATE scBNDtl SET TrackDelivered = 1 WHERE FID = @p1", dtlFID)
	if err != nil {
		return fmt.Errorf("标记签收失败 dtlFID=%d: %w", dtlFID, err)
	}
	log.Printf("标记签收: dtlFID=%d", dtlFID)
	return nil
}
