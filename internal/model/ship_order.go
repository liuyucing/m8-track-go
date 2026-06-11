package model

// ShipOrder 对应 Java ShipOrderDTO，查询 scbn + scBNDtl 的结果
type ShipOrder struct {
	FID   string `json:"fid" db:"FID"`
	MDNo  string `json:"mdNo" db:"MDNo"`
	FCKeY *int   `json:"fcKey" db:"FCkey"` // nullable，承运商代码
}
