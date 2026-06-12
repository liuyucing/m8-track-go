package model

import "time"

// TrackSyncRecord 对应 Java TrackSyncRecord，映射 track_sync_record 表
type TrackSyncRecord struct {
	ID            int64      `json:"id" db:"id"`
	FID           string     `json:"fid" db:"fid"`         // scbn.FID 表头单号（仅展示/兼容）
	DtlFID        *int64     `json:"dtlFid" db:"dtl_fid"`  // scBNDtl.FID 明细行主键，稳定身份；可空（旧记录未回填时为 nil）
	MDNo          string     `json:"mdNo" db:"md_no"`
	TrackStatus   *string    `json:"trackStatus" db:"track_status"`     // 17track 状态：InTransit/Delivered/...
	LastEvent     *string    `json:"lastEvent" db:"last_event"`         // 最新事件描述
	LastEventTime *time.Time `json:"lastEventTime" db:"last_event_time"`
	LastSyncTime  *time.Time `json:"lastSyncTime" db:"last_sync_time"`
	IsDelivered   bool       `json:"isDelivered" db:"is_delivered"`     // 0=未签收，1=已签收
	CreateTime    time.Time  `json:"createTime" db:"create_time"`
	UpdateTime    time.Time  `json:"updateTime" db:"update_time"`
}
