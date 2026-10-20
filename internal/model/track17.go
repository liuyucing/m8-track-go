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
	Number             string           `json:"number"`
	ShipDate           *string          `json:"ship_date,omitempty"`           // 顶层发货日期（YYYY-MM-DD），权威
	DestinationCountry string           `json:"destination_country,omitempty"` // 目的地国家代码
	TrackInfo          *TrackInfoDetail `json:"track_info,omitempty"`
}

// TrackInfoDetail 轨迹详情
type TrackInfoDetail struct {
	LatestStatus *LatestStatus  `json:"latest_status,omitempty"`
	LatestEvent  *TrackEvent    `json:"latest_event,omitempty"`
	ShippingInfo *ShippingInfo  `json:"shipping_info,omitempty"`
	Milestone    []Milestone    `json:"milestone,omitempty"`    // 关键里程碑（PickedUp/Delivered 等）
	Tracking     *TrackingBlock `json:"tracking,omitempty"`     // 各承运商的完整事件
	TimeMetrics  *TimeMetrics   `json:"time_metrics,omitempty"` // 预计送达等时间指标
}

// LatestStatus 最新物流状态
type LatestStatus struct {
	Status string `json:"status"` // NotFound/InTransit/Expired/PickedUp/Delivered
	Sub    string `json:"sub,omitempty"`
}

// TrackEvent 物流事件（同时用于 latest_event 与 providers[].events[]）
type TrackEvent struct {
	TimeISO     string `json:"time_iso,omitempty"`
	Description string `json:"description,omitempty"`
	Location    string `json:"location,omitempty"`
	Stage       string `json:"stage,omitempty"` // InfoReceived/PickedUp/InTransit/Arrived/OutForDelivery/Delivered/...
}

// ShippingInfo 寄件/收货地址信息
type ShippingInfo struct {
	RecipientAddress *Address `json:"recipient_address,omitempty"` // 收货地址（目的地）
}

// Address 地址
type Address struct {
	Country string `json:"country,omitempty"` // 国家代码，如 CN/AF/US
}

// Milestone 关键里程碑节点
type Milestone struct {
	KeyStage string  `json:"key_stage,omitempty"` // InfoReceived/PickedUp/Departure/Arrival/AvailableForPickup/OutForDelivery/Delivered/Returning/Returned
	TimeISO  *string `json:"time_iso,omitempty"`  // 指针：未来阶段为 null
}

// TrackingBlock 各承运商轨迹聚合
type TrackingBlock struct {
	Providers []Provider `json:"providers,omitempty"`
}

// Provider 单个承运商及其事件
type Provider struct {
	Provider ProviderName `json:"provider,omitempty"`
	Events   []TrackEvent `json:"events,omitempty"`
}

// ProviderName 承运商名称信息
type ProviderName struct {
	Name    string `json:"name,omitempty"`
	Country string `json:"country,omitempty"`
}

// TimeMetrics 时间指标
type TimeMetrics struct {
	EstimatedDeliveryDate *EDD `json:"estimated_delivery_date,omitempty"`
}

// EDD 预计送达日期区间
type EDD struct {
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
}
