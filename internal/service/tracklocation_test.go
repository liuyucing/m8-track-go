package service

import "testing"

func TestDeliveryLocation_LastEventPreferred(t *testing.T) {
	events := []NormalizedEvent{
		{Location: "YIWU, CN"},
		{Location: "SHANGHAI, CN"},
		{Location: "POTTSTOWN, PA, US"},
	}
	if got := DeliveryLocation(events); got != "POTTSTOWN, PA, US" {
		t.Fatalf("got %q, want POTTSTOWN, PA, US", got)
	}
}

func TestDeliveryLocation_FallbackToLastNonEmpty(t *testing.T) {
	events := []NormalizedEvent{
		{Location: "YIWU, CN"},
		{Location: ""},
	}
	if got := DeliveryLocation(events); got != "YIWU, CN" {
		t.Fatalf("got %q, want YIWU, CN", got)
	}
	if got := DeliveryLocation(nil); got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

func TestMatchLocation_RealCase(t *testing.T) {
	// 真实样例：FID=8234，17track 地点 vs 客户自由文本地址
	cust := "Keith Kovach\r\n610.544.8690\r\nJBS\r\n135 Walnut Street\r\nPottstown PA 19464"
	if got := MatchLocation(cust, "POTTSTOWN, PA, US"); got != LocationMatched {
		t.Fatalf("got %q, want %q", got, LocationMatched)
	}
}

func TestMatchLocation_ExactContains(t *testing.T) {
	if got := MatchLocation("ship to: pottstown, pa, us", "POTTSTOWN, PA, US"); got != LocationMatched {
		t.Fatalf("got %q, want %q", got, LocationMatched)
	}
}

func TestMatchLocation_CityMismatch(t *testing.T) {
	if got := MatchLocation("88 Market Street Boston MA 02108", "POTTSTOWN, PA, US"); got != LocationUnmatched {
		t.Fatalf("got %q, want %q", got, LocationUnmatched)
	}
}

func TestMatchLocation_EmptySide(t *testing.T) {
	if got := MatchLocation("", "POTTSTOWN, PA, US"); got != LocationUnmatched {
		t.Fatalf("got %q, want %q", got, LocationUnmatched)
	}
	if got := MatchLocation("some address", ""); got != LocationUnmatched {
		t.Fatalf("got %q, want %q", got, LocationUnmatched)
	}
}

func TestMatchLocation_CountryOnlyLocation(t *testing.T) {
	// 整串与分段都无法产生有意义比对 → 不匹配
	if got := MatchLocation("some address", "US"); got != LocationUnmatched {
		t.Fatalf("got %q, want %q", got, LocationUnmatched)
	}
}
