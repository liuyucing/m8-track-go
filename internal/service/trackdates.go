package service

import (
	"fmt"
	"strings"
	"time"

	"m8-track-go/internal/model"
)

// ExtractedDates 从 17track 轨迹推导出的发货/签收时间（按目的地时区，保留完整时分秒）。
type ExtractedDates struct {
	ShipDate      *time.Time // 发货完整时间（已签收→目的地时区；未签收→寄件地偏移）；顶层 ship_date 仅日期则 00:00
	DeliveredDate *time.Time // 签收完整时间（目的地时区）
	DestZoneName  string     // 命中的目的地时区名，用于日志
}

// ResolveDates 从 17track 轨迹信息推导发货/签收时间（保留完整时分秒，写入 ERP DATETIME）。
//
// 目的地时区优先取自 Delivered 里程碑的 time_iso 偏移（签收发生在目的地）；
// 未签收时回退到 defaultZone（nil 则用服务器本地时区）。
//
// 发货时间：顶层 ship_date 权威（仅有日期，时间恒为 00:00）；否则取 PickedUp 里程碑；
// 再否则取最早事件。已签收时换算到目的地时区；未签收时用事件自身（寄件地）偏移。
//
// 签收时间：仅当存在 Delivered 里程碑时取其完整时间（目的地时区）。
func ResolveDates(info *model.Track17TrackInfo, defaultZone *time.Location) ExtractedDates {
	out := ExtractedDates{}
	if info == nil || info.TrackInfo == nil {
		return out
	}

	destZone := defaultZone
	if destZone == nil {
		destZone = time.Local
	}

	deliveredT, delivered := findMilestoneTime(info.TrackInfo.Milestone, "Delivered")
	if delivered {
		if z := zoneFromTime(deliveredT); z != nil {
			destZone = z
			out.DestZoneName = z.String()
		}
		// deliveredT 自身偏移即目的地，保留完整时间写入
		d := deliveredT
		out.DeliveredDate = &d
	}

	out.ShipDate = resolveShipDate(info, delivered, destZone)
	return out
}

func resolveShipDate(info *model.Track17TrackInfo, delivered bool, destZone *time.Location) *time.Time {
	ti := info.TrackInfo

	// a) 顶层 ship_date（纯日期，不跨时区漂移）
	if info.ShipDate != nil && strings.TrimSpace(*info.ShipDate) != "" {
		if t, ok := parseDateOnly(*info.ShipDate); ok {
			d := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, destZone)
			return &d
		}
	}

	// b) PickedUp 里程碑
	if t, ok := findMilestoneTime(ti.Milestone, "PickedUp"); ok {
		return shipDateFromTime(t, delivered, destZone)
	}

	// c) 最早事件
	if events := CollectEvents(info); len(events) > 0 {
		return shipDateFromTime(events[0].Time, delivered, destZone)
	}
	return nil
}

func shipDateFromTime(t time.Time, delivered bool, destZone *time.Location) *time.Time {
	if delivered {
		// 已签收：换算到目的地时区，保留完整时间
		d := t.In(destZone)
		return &d
	}
	// 未签收：用事件自身（寄件地）偏移，保留完整时间
	d := t
	return &d
}

// findMilestoneTime 返回指定 key_stage 的首个非空可解析时间。
func findMilestoneTime(ms []model.Milestone, stage string) (time.Time, bool) {
	for _, m := range ms {
		if m.KeyStage != stage || m.TimeISO == nil {
			continue
		}
		if t, ok := parseISO(*m.TimeISO); ok {
			return t, true
		}
	}
	return time.Time{}, false
}

func zoneFromTime(t time.Time) *time.Location {
	_, off := t.Zone()
	return time.FixedZone(zoneName(off), off)
}

func zoneName(off int) string {
	sign := "+"
	if off < 0 {
		sign = "-"
	}
	a := off
	if a < 0 {
		a = -a
	}
	return fmt.Sprintf("UTC%s%02d:%02d", sign, a/3600, (a%3600)/60)
}

func parseISO(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, false
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func parseDateOnly(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}
