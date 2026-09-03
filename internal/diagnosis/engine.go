// Package diagnosis turns error groups, timelines and anomalies into an
// explainable root-cause verdict using a set of transparent rules.
//
// Rules are registered from the rules subpackage; the engine picks the
// strongest supported hypothesis and downgrades to "Insufficient evidence"
// when nothing clears the confidence threshold.
package diagnosis

import (
	"time"

	"github.com/faultlens/faultlens/internal/anomaly"
	"github.com/faultlens/faultlens/internal/grouping"
	"github.com/faultlens/faultlens/internal/model"
	"github.com/faultlens/faultlens/internal/timeline"
)

// InsufficientEvidenceThreshold is the minimum confidence a hypothesis needs
// to be reported as a root cause. Below it the engine reports
// "Insufficient evidence" while keeping the candidate's evidence for
// explainability.
const InsufficientEvidenceThreshold = 0.35

// DiagnosisContext carries all pre-computed analysis results to the rules.
// Rules must never re-parse raw logs; they consume these aggregates.
type DiagnosisContext struct {
	// ErrorGroups are the normalized error clusters.
	ErrorGroups []grouping.ErrorGroup
	// Timeline is the per-minute bucket series.
	Timeline []timeline.Bucket
	// Anomalies are the buckets flagged by the anomaly detector.
	Anomalies []anomaly.Detection
	// FiveXXCount / FiveXXFirst are the precomputed HTTP 5xx statistics,
	// computed once during streaming so rules never rescan stored events.
	FiveXXCount int
	FiveXXFirst time.Time
}

// DiagnosisRule evaluates one root-cause hypothesis against the context.
// Evaluate may return nil when the rule finds no evidence at all.
type DiagnosisRule interface {
	ID() string
	Evaluate(ctx *DiagnosisContext) *model.Diagnosis
}

// Engine runs all registered rules and selects the strongest hypothesis.
type Engine struct {
	rules []DiagnosisRule
}

// NewEngine returns an empty Engine. Register rules via AddRule, or call
// rules.RegisterDefaultRules for the built-in set.
func NewEngine() *Engine { return &Engine{} }

// AddRule registers a rule. Later rules win ties (stable priority).
func (e *Engine) AddRule(r DiagnosisRule) *Engine {
	e.rules = append(e.rules, r)
	return e
}

// RuleCount returns the number of registered rules.
func (e *Engine) RuleCount() int { return len(e.rules) }

// Diagnose evaluates every rule and returns the highest-confidence
// hypothesis. If nothing reaches InsufficientEvidenceThreshold, a
// low-confidence "Insufficient evidence" diagnosis is returned with the best
// candidate's evidence preserved so the verdict stays explainable.
func (e *Engine) Diagnose(ctx *DiagnosisContext) *model.Diagnosis {
	var best *model.Diagnosis
	for _, r := range e.rules {
		d := r.Evaluate(ctx)
		if d == nil {
			continue
		}
		d.Confidence = ClampConfidence(d.Confidence)
		if best == nil || d.Confidence > best.Confidence {
			best = d
		}
	}

	if best == nil {
		return &model.Diagnosis{
			RootCause:  "Insufficient evidence",
			Confidence: 0,
			Severity:   model.SeverityLow,
		}
	}
	if best.Confidence < InsufficientEvidenceThreshold {
		return &model.Diagnosis{
			RootCause:       "Insufficient evidence",
			Confidence:      best.Confidence,
			Severity:        best.Severity,
			Evidence:        best.Evidence,
			Recommendations: best.Recommendations,
		}
	}
	return best
}
