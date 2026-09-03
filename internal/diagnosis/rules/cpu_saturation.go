package rules

import (
	"github.com/faultlens/faultlens/internal/diagnosis"
	"github.com/faultlens/faultlens/internal/model"
	"github.com/faultlens/faultlens/internal/timeline"
)

// CPUSaturationRule detects sustained CPU saturation. Unlike keyword-only
// rules it also weighs the timeline: a strong temporal signal (errors rising
// in the second half of the window) strengthens the verdict.
type CPUSaturationRule struct{}

// NewCPUSaturationRule returns a CPUSaturationRule.
func NewCPUSaturationRule() *CPUSaturationRule { return &CPUSaturationRule{} }

// ID implements diagnosis.DiagnosisRule.
func (*CPUSaturationRule) ID() string { return "cpu_saturation" }

var cpuKeywords = []string{
	"cpu usage", "load average", "throttled", "cpu 100",
}

var cpuRecommendations = []string{
	"Check container CPU limits",
	"Inspect CPU usage by process",
	"Review recent traffic increase",
	"Check for infinite loops or hot code paths",
}

// Evaluate implements diagnosis.DiagnosisRule.
func (*CPUSaturationRule) Evaluate(ctx *diagnosis.DiagnosisContext) *model.Diagnosis {
	count, first, _ := diagnosis.CountKeywordGroups(ctx.ErrorGroups, cpuKeywords)
	if count == 0 {
		return nil
	}

	conf := diagnosis.ScoreStrong
	evidence := []model.Evidence{
		diagnosis.NewErrorPatternEvidence(first, "CPU saturation indicators detected", diagnosis.ScoreStrong),
	}

	// Time-series signal: errors in the second half of the timeline exceed
	// the first half, indicating sustained or escalating saturation. This
	// is the temporal evidence plan requires for CPU rules.
	if ratio := timeSeriesIncrease(ctx.Timeline); ratio >= 2 {
		conf += diagnosis.ScoreTemporal
		evidence = append(evidence,
			diagnosis.NewTemporalEvidence(first, "error volume increased over the analysis window", diagnosis.ScoreTemporal))
	}

	if len(ctx.Anomalies) > 0 {
		conf += diagnosis.ScoreAnomaly
		evidence = append(evidence,
			diagnosis.NewAnomalyEvidence(ctx.Anomalies[0].Bucket, "error rate anomaly detected", diagnosis.ScoreAnomaly))
	}

	return &model.Diagnosis{
		RootCause:       "CPU saturation",
		Confidence:      conf,
		Severity:        model.SeverityHigh,
		Evidence:        evidence,
		Recommendations: cpuRecommendations,
	}
}

// timeSeriesIncrease returns the ratio of error counts in the second half of
// the timeline to the first half. A ratio of 0 means no meaningful trend
// (empty or single-bucket timeline, or zero baseline).
func timeSeriesIncrease(tl []timeline.Bucket) float64 {
	n := len(tl)
	if n < 2 {
		return 0
	}
	half := n / 2
	var first, second int
	for i, b := range tl {
		if i < half {
			first += b.Errors
		} else {
			second += b.Errors
		}
	}
	if first == 0 {
		return 0
	}
	return float64(second) / float64(first)
}
