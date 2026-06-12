package model

// ShipOrder 对应 Java ShipOrderDTO，查询 scbn + scBNDtl 的结果
type ShipOrder struct {
	FID     string  `json:"fid" db:"FID"`
	DtlFID  int64   `json:"dtlFid" db:"DtlFID"`    // scBNDtl.FID，明细行主键（稳定身份）
	MDNo    string  `json:"mdNo" db:"MDNo"`
	FCKeY   *int    `json:"fcKey" db:"FCkey"`      // nullable，承运商代码
	OldMDNo *string `json:"oldMdNo" db:"OldMDNo"`  // 运单号变更前的旧号；仅当检测到该 FID 的 MDNo 变更时非 nil
}
