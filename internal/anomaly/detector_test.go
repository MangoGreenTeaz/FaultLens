package anomaly

import (
	"testing"
	"time"

	"github.com/MangoGreenTeaz/FaultLens/internal/timeline"
)

// bucket builds a timeline from per-bucket error counts starting at base.
func bucket(base time.Time, errors ...int) []timeline.Bucket {
	out := make([]timeline.Bucket, len(errors))
	for i, n := range errors {
		out[i] = timeline.Bucket{Start: base.Add(time.Duration(i) * time.Minute), Errors: n}
	}
	return out
}

func base() time.Time {
	return time.Date(2026, 8, 31, 14, 30, 0, 0, time.UTC)
}

func TestDetectNormalData(t *testing.T) {
	// Stable error counts must not be flagged.
	data := bucket(base(), 10, 12, 11, 13, 10, 12, 11, 12, 13, 11)
	detections := New().Detect(data)
	if len(detections) != 0 {
		t.Fatalf("got %d detections, want 0: %+v", len(detections), detections)
	}
}

func TestDetectClearSpike(t *testing.T) {
	// A strong spike after a stable baseline must be flagged with an
	// explainable baseline and increase.
	data := bucket(base(), 1, 2, 1, 2, 1, 2, 1, 2, 1, 50)
	detections := New().Detect(data)
	if len(detections) != 1 {
		t.Fatalf("got %d detections, want 1: %+v", len(detections), detections)
	}
	d := detections[0]
	if !d.Bucket.Equal(base().Add(9 * time.Minute)) {
		t.Errorf("Bucket = %v, want the 10th bucket", d.Bucket)
	}
	if d.Current != 50 {
		t.Errorf("Current = %d, want 50", d.Current)
	}
	if d.BaselineMean <= 0 {
		t.Errorf("BaselineMean = %v, want > 0", d.BaselineMean)
	}
	if d.Increase < 10 {
		t.Errorf("Increase = %v, want >> 3", d.Increase)
	}
	if d.ZScore < 3 {
		t.Errorf("ZScore = %v, want >= 3", d.ZScore)
	}
}

func TestDetectSmallSampleNotFlagged(t *testing.T) {
	// Fewer buckets than MinBaseline: even a spike must not be flagged.
	data := bucket(base(), 1, 2, 3, 4, 50)
	detections := New().Detect(data)
	if len(detections) != 0 {
		t.Fatalf("got %d detections, want 0 (small sample protection)", len(detections))
	}
}

func TestDetectConstantDataNotFlagged(t *testing.T) {
	// Constant error counts (std == 0) and no increase: nothing flagged.
	data := bucket(base(), 5, 5, 5, 5, 5, 5, 5, 5, 5, 5)
	detections := New().Detect(data)
	if len(detections) != 0 {
		t.Fatalf("got %d detections, want 0", len(detections))
	}
}

func TestDetectConstantBaselineWithIncrease(t *testing.T) {
	// Constant baseline (std == 0) with a strong increase falls back to the
	// multiple rule and is still flagged.
	data := bucket(base(), 5, 5, 5, 5, 5, 5, 5, 5, 5, 25)
	detections := New().Detect(data)
	if len(detections) != 1 {
		t.Fatalf("got %d detections, want 1: %+v", len(detections), detections)
	}
	if detections[0].Increase != 5 {
		t.Errorf("Increase = %v, want 5", detections[0].Increase)
	}
}

func TestDetectSingleBucketNeverFlagged(t *testing.T) {
	data := bucket(base(), 100)
	if detections := New().Detect(data); len(detections) != 0 {
		t.Fatalf("single bucket flagged: %+v", detections)
	}
}

func TestDetectLowVolumeNotFlagged(t *testing.T) {
	// Current bucket below MinErrors must not be flagged even with a huge
	// relative increase (low-traffic false-positive protection).
	data := bucket(base(), 1, 1, 1, 1, 1, 1, 1, 1, 1, 8)
	detections := New().Detect(data)
	if len(detections) != 0 {
		t.Fatalf("got %d detections, want 0 (MinErrors gate)", len(detections))
	}
}

func TestDetectCustomThreshold(t *testing.T) {
	// A tighter increase threshold should flag a bucket the default rejects.
	cfg := Config{MinIncrease: 1.5}
	// Modest increase (2x) with a constant baseline: flagged with looser
	// threshold, not with the default.
	data := bucket(base(), 5, 5, 5, 5, 5, 5, 5, 5, 5, 10)
	loose := NewWithConfig(cfg).Detect(data)
	if len(loose) != 1 {
		t.Fatalf("got %d detections with loose threshold, want 1", len(loose))
	}
	if got := New().Detect(data); len(got) != 0 {
		t.Fatalf("default threshold flagged 2x increase: %+v", got)
	}
}

func TestDetectEmptyInput(t *testing.T) {
	if detections := New().Detect(nil); len(detections) != 0 {
		t.Fatalf("empty input produced detections: %+v", detections)
	}
}

func TestConfigDefaults(t *testing.T) {
	d := New()
	if d.cfg.MinBaseline != DefaultMinBaseline {
		t.Errorf("MinBaseline = %d, want %d", d.cfg.MinBaseline, DefaultMinBaseline)
	}
	if d.cfg.ZScore != DefaultZScore {
		t.Errorf("ZScore = %v, want %v", d.cfg.ZScore, DefaultZScore)
	}
	if d.cfg.MinIncrease != DefaultMinIncrease {
		t.Errorf("MinIncrease = %v, want %v", d.cfg.MinIncrease, DefaultMinIncrease)
	}
	if d.cfg.MinErrors != DefaultMinErrors {
		t.Errorf("MinErrors = %d, want %d", d.cfg.MinErrors, DefaultMinErrors)
	}
}
