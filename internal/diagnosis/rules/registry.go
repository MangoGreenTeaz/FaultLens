// Package rules implements the six built-in root-cause hypotheses.
package rules

import (
	"github.com/faultlens/faultlens/internal/diagnosis"
)

// Shared evidence thresholds used across rules. Kept in one place so tuning
// the sensitivity of the scoring system is a single-point change.
const (
	// strongEvidenceThreshold is the group count that upgrades evidence to
	// "strong" (+ScoreStrong instead of +ScoreSupporting).
	strongEvidenceThreshold = 10
	// largeVolumeThreshold marks sustained, high-volume error patterns.
	largeVolumeThreshold = 20
	// crashVolumeThreshold marks repeated crash indicators.
	crashVolumeThreshold = 5
)

// RegisterDefaultRules wires the six built-in rules into the engine.
// Later registrations win ties, so the order below defines the priority for
// equal-confidence hypotheses.
func RegisterDefaultRules(e *diagnosis.Engine) *diagnosis.Engine {
	e.AddRule(NewDatabaseRule())
	e.AddRule(NewRedisRule())
	e.AddRule(NewOOMRule())
	e.AddRule(NewTimeoutRule())
	e.AddRule(NewHTTPRule())
	e.AddRule(NewCrashRule())
	return e
}
