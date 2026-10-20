package service

import (
	"testing"
	"time"

	"m8-track-go/internal/model"
)

func TestCollectEvents_MultiProviderMergeSorted(t *testing.T) {
	info := &model.Track17TrackInfo{TrackInfo: &model.TrackInfoDetail{
		Tracking: &model.TrackingBlock{Providers: []model.Provider{
			{Provider: model.ProviderName{Name: "FedEx"}, Events: []model.TrackEvent{
				{TimeISO: "2022-03-02T10:00:00+00:00", Description: "b"},
			}},
			{Provider: model.ProviderName{Name: "Australia Post"}, Events: []model.TrackEvent{
				{TimeISO: "2022-03-01T10:00:00+00:00", Description: "a"},
				{TimeISO: "not-a-date", Description: "dropped"},
			}},
		}},
	}}
	ev := CollectEvents(info)
	if len(ev) != 2 {
		t.Fatalf("got %d events, want 2 (bad iso dropped)", len(ev))
	}
	if ev[0].Desc != "a" || ev[1].Desc != "b" {
		t.Fatalf("order wrong: %s then %s, want a then b", ev[0].Desc, ev[1].Desc)
	}
}

func TestCollectEvents_NilSafe(t *testing.T) {
	if got := CollectEvents(nil); got != nil {
		t.Fatalf("CollectEvents(nil) = %v, want nil", got)
	}
	if got := CollectEvents(&model.Track17TrackInfo{}); got != nil {
		t.Fatalf("CollectEvents(empty) = %v, want nil", got)
	}
}

func TestEventDedupKey_SecondsJitterSameKey(t *testing.T) {
	a, ok := parseISO("2022-03-01T10:00:05+00:00")
	if !ok {
		t.Fatal("parse failed")
	}
	b, ok := parseISO("2022-03-01T10:00:29+00:00")
	if !ok {
		t.Fatal("parse failed")
	}
	e1 := NormalizedEvent{Time: a, Stage: "InTransit", Desc: "x"}
	e2 := NormalizedEvent{Time: b, Stage: "InTransit", Desc: "x"}
	if eventDedupKey(e1) != eventDedupKey(e2) {
		t.Fatalf("expected same dedup key for same-minute events")
	}
}

func TestEventDedupKey_DifferentDescDifferentKey(t *testing.T) {
	a, _ := parseISO("2022-03-01T10:00:00+00:00")
	e1 := NormalizedEvent{Time: a, Stage: "InTransit", Desc: "x"}
	e2 := NormalizedEvent{Time: a, Stage: "InTransit", Desc: "y"}
	if eventDedupKey(e1) == eventDedupKey(e2) {
		t.Fatalf("expected different keys for different descriptions")
	}
}

// TestEventDedupKey_DBWriteReadBackSameKey 复现历史重复入库缺陷：
// 17track 返回带 +08:00 偏移的 time_iso；驱动写入 DATETIME 列丢弃偏移只留墙上时钟，
// 读回是"伪装成 UTC 的本地时间"。去重键必须让两种形态得到同一个键。
func TestEventDedupKey_DBWriteReadBackSameKey(t *testing.T) {
	fresh, ok := parseISO("2026-10-13T19:48:00+08:00")
	if !ok {
		t.Fatal("parse failed")
	}
	// 模拟 DATETIME 读回：墙上时钟不变，位置变成 UTC
	readBack := time.Date(2026, 10, 13, 19, 48, 0, 0, time.UTC)
	e1 := NormalizedEvent{Time: fresh, Stage: "PickedUp", Desc: "Picked up"}
	e2 := NormalizedEvent{Time: readBack, Stage: "PickedUp", Desc: "Picked up"}
	if eventDedupKey(e1) != eventDedupKey(e2) {
		t.Fatalf("dedup key mismatch after DB round-trip: fresh=%q readBack=%q",
			eventDedupKey(e1), eventDedupKey(e2))
	}
}
