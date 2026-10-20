package service

import (
	"testing"
	"time"

	"m8-track-go/internal/model"
)

func strPtr(s string) *string { return &s }

func mkMilestone(stage, iso string) model.Milestone {
	m := model.Milestone{KeyStage: stage}
	if iso != "" {
		m.TimeISO = strPtr(iso)
	}
	return m
}

func dateOnly(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format("2006-01-02")
}

func TestResolveDates_DeliveredProvidesDestZone(t *testing.T) {
	info := &model.Track17TrackInfo{TrackInfo: &model.TrackInfoDetail{
		Milestone: []model.Milestone{
			mkMilestone("PickedUp", "2022-03-01T20:00:00-06:00"),
			mkMilestone("Delivered", "2022-03-10T12:00:00-05:00"),
		},
	}}
	got := ResolveDates(info, time.UTC)
	if dateOnly(got.DeliveredDate) != "2022-03-10" {
		t.Fatalf("delivered_date = %q, want 2022-03-10", dateOnly(got.DeliveredDate))
	}
	// PickedUp 20:00 -06:00 → instant 2022-03-02T02:00Z → in -05:00 = 2022-03-01T19:00 → 2022-03-01
	if dateOnly(got.ShipDate) != "2022-03-01" {
		t.Fatalf("ship_date = %q, want 2022-03-01", dateOnly(got.ShipDate))
	}
	if got.DestZoneName != "UTC-05:00" {
		t.Fatalf("dest zone = %q, want UTC-05:00", got.DestZoneName)
	}
}

func TestResolveDates_NotDelivered_UsesOriginOffsetForShip(t *testing.T) {
	info := &model.Track17TrackInfo{TrackInfo: &model.TrackInfoDetail{
		Milestone: []model.Milestone{
			mkMilestone("PickedUp", "2022-03-01T20:00:00-06:00"),
		},
	}}
	got := ResolveDates(info, time.UTC)
	if got.DeliveredDate != nil {
		t.Fatalf("delivered_date = %v, want nil", got.DeliveredDate)
	}
	if dateOnly(got.ShipDate) != "2022-03-01" {
		t.Fatalf("ship_date = %q, want 2022-03-01 (寄件地 -06:00)", dateOnly(got.ShipDate))
	}
}

func TestResolveDates_TopLevelShipDateAuthoritative(t *testing.T) {
	info := &model.Track17TrackInfo{
		ShipDate: strPtr("2022-03-01"),
		TrackInfo: &model.TrackInfoDetail{
			Milestone: []model.Milestone{
				mkMilestone("PickedUp", "2022-04-05T10:00:00+08:00"),
				mkMilestone("Delivered", "2022-04-10T10:00:00+08:00"),
			},
		},
	}
	got := ResolveDates(info, time.UTC)
	if dateOnly(got.ShipDate) != "2022-03-01" {
		t.Fatalf("ship_date = %q, want 2022-03-01 (顶层权威)", dateOnly(got.ShipDate))
	}
}

func TestResolveDates_PickedUpMissing_FallbackEarliestEvent(t *testing.T) {
	info := &model.Track17TrackInfo{TrackInfo: &model.TrackInfoDetail{
		Tracking: &model.TrackingBlock{Providers: []model.Provider{
			{Events: []model.TrackEvent{
				{TimeISO: "2022-05-02T09:00:00+00:00", Description: "in transit"},
				{TimeISO: "2022-05-01T09:00:00+00:00", Description: "accepted"},
			}},
		}},
	}}
	got := ResolveDates(info, time.UTC)
	// 未签收 → 用最早事件自身偏移（+00:00）→ 2022-05-01
	if dateOnly(got.ShipDate) != "2022-05-01" {
		t.Fatalf("ship_date = %q, want 2022-05-01", dateOnly(got.ShipDate))
	}
}

func TestResolveDates_NullDeliveredMilestoneSkipped(t *testing.T) {
	info := &model.Track17TrackInfo{TrackInfo: &model.TrackInfoDetail{
		Milestone: []model.Milestone{
			mkMilestone("PickedUp", "2022-03-01T20:00:00-06:00"),
			mkMilestone("Delivered", ""), // nil → 未来阶段
		},
	}}
	got := ResolveDates(info, time.UTC)
	if got.DeliveredDate != nil {
		t.Fatalf("delivered_date = %v, want nil", got.DeliveredDate)
	}
}

func TestResolveDates_MalformedISOReturnsNil(t *testing.T) {
	info := &model.Track17TrackInfo{TrackInfo: &model.TrackInfoDetail{
		Milestone: []model.Milestone{
			mkMilestone("Delivered", "not-a-date"),
		},
	}}
	got := ResolveDates(info, time.UTC)
	if got.DeliveredDate != nil {
		t.Fatalf("delivered_date = %v, want nil on malformed iso", got.DeliveredDate)
	}
}

func TestResolveDates_CrossMidnight(t *testing.T) {
	info := &model.Track17TrackInfo{TrackInfo: &model.TrackInfoDetail{
		Milestone: []model.Milestone{
			mkMilestone("PickedUp", "2022-03-01T23:30:00-06:00"),
			mkMilestone("Delivered", "2022-03-12T12:00:00-05:00"),
		},
	}}
	got := ResolveDates(info, time.UTC)
	// PickedUp 23:30 -06:00 → instant 2022-03-02T05:30Z → in -05:00 = 2022-03-02T00:30 → 2022-03-02
	if dateOnly(got.ShipDate) != "2022-03-02" {
		t.Fatalf("ship_date = %q, want 2022-03-02 (跨午夜)", dateOnly(got.ShipDate))
	}
}

func TestResolveDates_PreservesFullTime(t *testing.T) {
	info := &model.Track17TrackInfo{TrackInfo: &model.TrackInfoDetail{
		Milestone: []model.Milestone{
			mkMilestone("PickedUp", "2022-03-01T20:00:00-06:00"),
			mkMilestone("Delivered", "2022-03-10T12:34:56-05:00"),
		},
	}}
	got := ResolveDates(info, time.UTC)
	// 签收保留完整时分秒（目的地 -05:00 墙钟），不再截断为 00:00:00
	if got.DeliveredDate == nil || got.DeliveredDate.Format("2006-01-02 15:04:05") != "2022-03-10 12:34:56" {
		t.Fatalf("delivered = %v, want 2022-03-10 12:34:56 (保留完整时间)", got.DeliveredDate)
	}
	// 发货：PickedUp 20:00-06:00 → 瞬间 2022-03-02T02:00Z → 换算到目的地 -05:00 墙钟 = 2022-03-01 21:00:00
	if got.ShipDate == nil || got.ShipDate.Format("2006-01-02 15:04:05") != "2022-03-01 21:00:00" {
		t.Fatalf("ship = %v, want 2022-03-01 21:00:00 (保留完整时间)", got.ShipDate)
	}
}

func TestResolveDates_TopLevelShipDateHasNoTime(t *testing.T) {
	// 顶层 ship_date 仅有日期 → 时间恒为 00:00:00
	info := &model.Track17TrackInfo{
		ShipDate: strPtr("2022-03-01"),
		TrackInfo: &model.TrackInfoDetail{
			Milestone: []model.Milestone{
				mkMilestone("Delivered", "2022-03-10T12:00:00-05:00"),
			},
		},
	}
	got := ResolveDates(info, time.UTC)
	if got.ShipDate == nil || got.ShipDate.Format("2006-01-02 15:04:05") != "2022-03-01 00:00:00" {
		t.Fatalf("ship = %v, want 2022-03-01 00:00:00 (顶层 ship_date 无时间)", got.ShipDate)
	}
}
