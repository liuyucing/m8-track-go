package service

import "strings"

// LocationMatchValue 匹配/不匹配 的字面值，写入 scBNDtl.locationmatch。
const (
	LocationMatched   = "匹配"
	LocationUnmatched = "不匹配"
)

// DeliveryLocation 取签收地点：轨迹事件按时间升序传入时，签收事件（最后一条）
// 的 location 即目的地；末条 location 为空则向前回溯最近一个非空地点。
// events 为空返回空串。
func DeliveryLocation(events []NormalizedEvent) string {
	for i := len(events) - 1; i >= 0; i-- {
		if loc := strings.TrimSpace(events[i].Location); loc != "" {
			return loc
		}
	}
	return ""
}

// MatchLocation 判断客户收货信息（scBNDtl.CustShipInfo）是否包含承运商地点
//（如 "POTTSTOWN, PA, US"），匹配返回"匹配"，否则"不匹配"。
//
// 两边格式天然不同（17track 为「城市, 州, 国家代码」，CustShipInfo 为自由文本地址），
// 因此先做忽略大小写的整串包含；整串未命中时退化为分段匹配：
// 按逗号拆分 location、剔除纯国家代码段，剩余段（城市、州）全部
// 忽略大小写出现在 CustShipInfo 中即视为匹配。
// 任一侧为空返回"不匹配"。
func MatchLocation(custShipInfo, location string) string {
	cust := normalizeForMatch(custShipInfo)
	loc := strings.TrimSpace(location)
	if cust == "" || loc == "" {
		return LocationUnmatched
	}

	if strings.Contains(cust, normalizeForMatch(loc)) {
		return LocationMatched
	}

	parts := strings.Split(loc, ",")
	sawMeaningful := false
	for _, p := range parts {
		p = normalizeForMatch(p)
		if p == "" || len(p) == 2 { // 两位即国家代码（US/CN/DE…），地址文本里通常不出现
			continue
		}
		if !strings.Contains(cust, p) {
			return LocationUnmatched
		}
		sawMeaningful = true
	}
	if !sawMeaningful {
		// 整串未命中且没有可用于分段比对的段（如仅国家代码），按不匹配处理
		return LocationUnmatched
	}
	return LocationMatched
}

// normalizeForMatch 小写化并去掉空白、逗号、点、换行，使自由文本地址可比较。
func normalizeForMatch(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		switch r {
		case ' ', '\t', '\r', '\n', ',', '.', ';', '、', '，':
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
