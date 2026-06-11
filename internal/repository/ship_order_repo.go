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

// SelectPendingOrders 查询未签收的待处理运单
// SQL 与 Java ShipOrderMapper.selectPendingOrders 完全一致
func (r *ShipOrderRepo) SelectPendingOrders(ctx context.Context) ([]model.ShipOrder, error) {
	query := `SELECT m.FID, trim(d1.MDNo) AS MDNo, d1.FCkey
              FROM scbn m
              INNER JOIN scBNDtl d1 ON m.fid = d1.mfid
              WHERE m.IsDeleted = 0
                AND m.CreateDate > @p1
                AND d1.MDNo IS NOT NULL
                AND d1.fckey IS NOT NULL
                AND d1.TrackDelivered = 0`

	rows, err := r.db.QueryContext(ctx, query, r.dateFilter+" 00:00:00")
	if err != nil {
		return nil, fmt.Errorf("查询待处理运单失败: %w", err)
	}
	defer rows.Close()

	var orders []model.ShipOrder
	for rows.Next() {
		var o model.ShipOrder
		if err := rows.Scan(&o.FID, &o.MDNo, &o.FCKeY); err != nil {
			return nil, fmt.Errorf("扫描运单数据失败: %w", err)
		}
		o.MDNo = strings.TrimSpace(o.MDNo)
		orders = append(orders, o)
	}
	return orders, rows.Err()
}

// UpdateFCtrack 更新运单的 FCtrack 字段
func (r *ShipOrderRepo) UpdateFCtrack(ctx context.Context, mdNo, value string) error {
	mdNo = strings.TrimSpace(mdNo)
	_, err := r.db.ExecContext(ctx, "UPDATE scBNDtl SET FCtrack = @p1 WHERE trim(MDNo) = trim(@p2)", value, mdNo)
	if err != nil {
		return fmt.Errorf("更新 FCtrack 失败 mdNo=%s: %w", mdNo, err)
	}
	log.Printf("更新 FCtrack: mdNo=%s, value=%s", mdNo, value)
	return nil
}

// MarkDelivered 标记运单为已签收
func (r *ShipOrderRepo) MarkDelivered(ctx context.Context, mdNo string) error {
	mdNo = strings.TrimSpace(mdNo)
	_, err := r.db.ExecContext(ctx, "UPDATE scBNDtl SET TrackDelivered = 1 WHERE trim(MDNo) = trim(@p1)", mdNo)
	if err != nil {
		return fmt.Errorf("标记签收失败 mdNo=%s: %w", mdNo, err)
	}
	log.Printf("标记签收: mdNo=%s", mdNo)
	return nil
}
