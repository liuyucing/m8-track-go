package model

import "time"

// TrackSyncDetail 对应 Java TrackSyncDetail，映射 track_sync_detail 表
type TrackSyncDetail struct {
	ID          int64      `json:"id" db:"id"`
	MDNo        string     `json:"mdNo" db:"md_no"`
	TrackStatus *string    `json:"trackStatus" db:"track_status"`
	EventDesc   *string    `json:"eventDesc" db:"event_desc"`
	EventTime   *time.Time `json:"eventTime" db:"event_time"`
	CreateTime  time.Time  `json:"createTime" db:"create_time"`
}
