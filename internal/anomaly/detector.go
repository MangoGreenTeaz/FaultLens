// Package anomaly detects statistically unusual error counts in time buckets
// using an explainable baseline: the mean and standard deviation of all
// previous buckets. No machine learning is involved.
package anomaly

import (
	"math"
	"time"

	"github.com/faultlens/faultlens/internal/timeline"
)

// Default configuration values.
const (
	DefaultMinBaseline = 5   // buckets required before any detection
	DefaultZScore      = 3.0 // z-score threshold
	DefaultMinIncrease = 3.0 // minimum multiple over the baseline mean
	DefaultMinErrors   = 10  // minimum current-bucket error count
)

// Config controls detection sensitivity. Zero values fall back to defaults.
type Config struct {
	// MinBaseline is how many previous buckets are required to form a
	// baseline. Fewer buckets → no detections (small-sample protection).
	MinBaseline int
	// ZScore is the z-score threshold above which a bucket is candidate.
	ZScore float64
	// MinIncrease is the minimum multiple over the baseline mean.
	MinIncrease float64
	// MinErrors is the minimum error count a current bucket must have.
	// Keeps low-traffic logs from producing false positives.
	MinErrors int
}

// Detection describes one anomalous bucket.
type Detection struct {
	Bucket       time.Time
	BaselineMean float64
	BaselineStd  float64
	Current      int
	ZScore       float64
	Increase     float64
}

// Detector finds anomalous buckets in a timeline.
type Detector struct {
	cfg Config
}

// New returns a Detector with default sensitivity.
func New() *Detector { return NewWithConfig(Config{}) }

// NewWithConfig returns a Detector with the given configuration. Zero-valued
// fields use the defaults.
func NewWithConfig(cfg Config) *Detector {
	if cfg.MinBaseline <= 0 {
		cfg.MinBaseline = DefaultMinBaseline
	}
	if cfg.ZScore <= 0 {
		cfg.ZScore = DefaultZScore
	}
	if cfg.MinIncrease <= 0 {
		cfg.MinIncrease = DefaultMinIncrease
	}
	if cfg.MinErrors <= 0 {
		cfg.MinErrors = DefaultMinErrors
	}
	return &Detector{cfg: cfg}
}

// Detect returns every bucket whose error count deviates strongly from the
// mean of the buckets that preceded it. The baseline for bucket i is buckets
// [0, i); the first MinBaseline buckets are never flagged.
func (d *Detector) Detect(buckets []timeline.Bucket) []Detection {
	var out []Detection

	for i := range buckets {
		if i < d.cfg.MinBaseline {
			continue
		}

		mean, std := baselineStats(buckets[:i])
		if mean <= 0 {
			continue
		}
		cur := buckets[i].Errors
		if cur < d.cfg.MinErrors {
			continue
		}

		increase := float64(cur) / mean
		det := Detection{
			Bucket:       buckets[i].Start,
			BaselineMean: mean,
			BaselineStd:  std,
			Current:      cur,
			Increase:     increase,
		}

		if std > 0 {
			det.ZScore = (float64(cur) - mean) / std
			if det.ZScore >= d.cfg.ZScore && increase >= d.cfg.MinIncrease {
				out = append(out, det)
			}
			continue
		}

		// Constant baseline (std == 0): fall back to the increase
		// multiple, which is still fully explainable.
		if increase >= d.cfg.MinIncrease {
			det.ZScore = math.Inf(1)
			out = append(out, det)
		}
	}

	return out
}

// baselineStats returns the mean and sample standard deviation of error
// counts. Sample (N-1) deviation is used so small baselines are conservative.
func baselineStats(buckets []timeline.Bucket) (mean, std float64) {
	n := len(buckets)
	if n == 0 {
		return 0, 0
	}
	var sum int
	for _, b := range buckets {
		sum += b.Errors
	}
	mean = float64(sum) / float64(n)
	if n < 2 {
		return mean, 0
	}
	var sq float64
	for _, b := range buckets {
		diff := float64(b.Errors) - mean
		sq += diff * diff
	}
	std = math.Sqrt(sq / float64(n-1))
	return mean, std
}
