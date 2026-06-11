package model

import "encoding/json"

// Track17RegisterReq 17track 注册请求体
type Track17RegisterReq struct {
	Number  string `json:"number"`
	Carrier int    `json:"carrier"` // 0=自动识别
}

// Track17Resp 17track API 通用响应
type Track17Resp struct {
	Code int             `json:"code"`
	Data json.RawMessage `json:"data,omitempty"`
}

// Track17RegisterData /register 响应的 data 部分
type Track17RegisterData struct {
	Accepted []Track17AcceptedItem `json:"accepted,omitempty"`
	Rejected []json.RawMessage     `json:"rejected,omitempty"`
}

// Track17AcceptedItem 注册成功的运单项
type Track17AcceptedItem struct {
	Number string `json:"number"`
}

// Track17RejectedItem 注册被拒绝的运单项
type Track17RejectedItem struct {
	Number string           `json:"number"`
	Error  Track17ErrorInfo `json:"error"`
}

// Track17ErrorInfo 17track 错误信息
type Track17ErrorInfo struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// Track17TrackInfo 查询轨迹返回的单条运单信息
type Track17TrackInfo struct {
	Number    string          `json:"number"`
	TrackInfo *TrackInfoDetail `json:"track_info,omitempty"`
}

// TrackInfoDetail 轨迹详情
type TrackInfoDetail struct {
	LatestStatus *LatestStatus `json:"latest_status,omitempty"`
	LatestEvent  *TrackEvent   `json:"latest_event,omitempty"`
}

// LatestStatus 最新物流状态
type LatestStatus struct {
	Status string `json:"status"` // NotFound/InTransit/Expired/PickedUp/Delivered
	Sub    string `json:"sub,omitempty"`
}

// TrackEvent 物流事件
type TrackEvent struct {
	TimeISO     string `json:"time_iso,omitempty"`
	Description string `json:"description,omitempty"`
	Location    string `json:"location,omitempty"`
}
