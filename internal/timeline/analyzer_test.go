package timeline

import (
	"testing"
	"time"

	"github.com/MangoGreenTeaz/FaultLens/internal/model"
)

func ts(h, m, s int) time.Time {
	return time.Date(2026, 8, 31, h, m, s, 0, time.UTC)
}

func ev(level model.LogLevel, at time.Time) *model.LogEvent {
	return &model.LogEvent{Level: level, Timestamp: at}
}

func TestAnalyzerCountsByLevel(t *testing.T) {
	a := New()
	base := ts(14, 32, 0)
	a.Add(ev(model.LevelInfo, base))
	a.Add(ev(model.LevelError, base.Add(10*time.Second)))
	a.Add(ev(model.LevelError, base.Add(20*time.Second)))
	a.Add(ev(model.LevelWarn, base.Add(30*time.Second)))
	a.Add(ev(model.LevelFatal, base.Add(40*time.Second)))

	buckets := a.Buckets()
	if len(buckets) != 1 {
		t.Fatalf("got %d buckets, want 1", len(buckets))
	}
	b := buckets[0]
	if b.Total != 5 {
		t.Errorf("Total = %d, want 5", b.Total)
	}
	if b.Errors != 2 {
		t.Errorf("Errors = %d, want 2", b.Errors)
	}
	if b.Warnings != 1 {
		t.Errorf("Warnings = %d, want 1", b.Warnings)
	}
	if b.Fatal != 1 {
		t.Errorf("Fatal = %d, want 1", b.Fatal)
	}
}

func TestAnalyzerBucketsAlignedToMinute(t *testing.T) {
	a := New()
	a.Add(ev(model.LevelError, ts(14, 32, 59)))
	a.Add(ev(model.LevelError, ts(14, 33, 0)))

	buckets := a.Buckets()
	if len(buckets) != 2 {
		t.Fatalf("got %d buckets, want 2", len(buckets))
	}
	if !buckets[0].Start.Equal(ts(14, 32, 0)) {
		t.Errorf("bucket[0] start = %v, want 14:32:00", buckets[0].Start)
	}
	if !buckets[1].Start.Equal(ts(14, 33, 0)) {
		t.Errorf("bucket[1] start = %v, want 14:33:00", buckets[1].Start)
	}
}

func TestAnalyzerSortsBuckets(t *testing.T) {
	a := New()
	a.Add(ev(model.LevelError, ts(14, 35, 0)))
	a.Add(ev(model.LevelError, ts(14, 30, 0)))
	a.Add(ev(model.LevelError, ts(14, 32, 0)))

	buckets := a.Buckets()
	if len(buckets) != 3 {
		t.Fatalf("got %d buckets, want 3", len(buckets))
	}
	for i := 1; i < len(buckets); i++ {
		if !buckets[i-1].Start.Before(buckets[i].Start) {
			t.Errorf("buckets not sorted: %v before %v", buckets[i-1].Start, buckets[i].Start)
		}
	}
}

func TestAnalyzerIgnoresZeroTimestamp(t *testing.T) {
	a := New()
	a.Add(ev(model.LevelError, time.Time{}))
	a.Add(nil)
	if len(a.Buckets()) != 0 {
		t.Errorf("zero-timestamp / nil events must be ignored")
	}
}

func TestAnalyzerCustomInterval(t *testing.T) {
	a := NewWithInterval(5 * time.Minute)
	base := ts(14, 30, 0)
	a.Add(ev(model.LevelError, base))
	a.Add(ev(model.LevelError, base.Add(4*time.Minute)))
	a.Add(ev(model.LevelError, base.Add(5*time.Minute)))

	if a.Interval() != 5*time.Minute {
		t.Errorf("Interval() = %v, want 5m", a.Interval())
	}
	buckets := a.Buckets()
	if len(buckets) != 2 {
		t.Fatalf("got %d buckets, want 2 (4m in same 5m bucket, 5m in next)", len(buckets))
	}
	if buckets[0].Total != 2 {
		t.Errorf("bucket[0] Total = %d, want 2", buckets[0].Total)
	}
	if buckets[1].Total != 1 {
		t.Errorf("bucket[1] Total = %d, want 1", buckets[1].Total)
	}
}

func TestAnalyzerDefaultInterval(t *testing.T) {
	if New().Interval() != DefaultInterval {
		t.Errorf("default interval = %v, want %v", New().Interval(), DefaultInterval)
	}
}
