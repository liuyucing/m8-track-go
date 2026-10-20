package service

import (
	"sort"
	"time"

	"m8-track-go/internal/model"
)

// NormalizedEvent 归一化后的轨迹事件，用于持久化与去重。
type NormalizedEvent struct {
	Time     time.Time
	Desc     string
	Stage    string
	Location string
	Provider string
}

// CollectEvents 汇总 17track 响应里全部承运商的轨迹事件，
// 丢弃 time_iso 为空或不可解析的事件，按时间升序返回。
func CollectEvents(info *model.Track17TrackInfo) []NormalizedEvent {
	if info == nil || info.TrackInfo == nil || info.TrackInfo.Tracking == nil {
		return nil
	}
	var events []NormalizedEvent
	for _, p := range info.TrackInfo.Tracking.Providers {
		name := p.Provider.Name
		for _, e := range p.Events {
			t, ok := parseISO(e.TimeISO)
			if !ok {
				continue
			}
			events = append(events, NormalizedEvent{
				Time:     t,
				Desc:     e.Description,
				Stage:    e.Stage,
				Location: e.Location,
				Provider: name,
			})
		}
	}
	sort.SliceStable(events, func(i, j int) bool {
		return events[i].Time.Before(events[j].Time)
	})
	return events
}

// eventDedupKey 事件去重键：(本地墙上时钟分钟, stage, description)。
// 分钟截断吸收 17track 跨同步的秒级抖动。
//
// 必须用事件自身时区的墙上时钟（不得转 UTC）：time_iso 带 +08:00 这类偏移，
// 而驱动写入 DATETIME 列时丢弃偏移只留墙上时钟，读回是"伪装成 UTC 的本地时间"。
// 若统一转 UTC 比较，写入方是 +08 墙上时钟、读取方被再转一次 UTC，键永远对不上，
// 导致每次同步全量重复入库。
func eventDedupKey(e NormalizedEvent) string {
	minute := e.Time.Truncate(time.Minute).Format("2006-01-02T15:04")
	return minute + "|" + e.Stage + "|" + e.Desc
}
