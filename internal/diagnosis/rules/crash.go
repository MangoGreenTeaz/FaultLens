package rules

import (
	"github.com/faultlens/faultlens/internal/diagnosis"
	"github.com/faultlens/faultlens/internal/model"
)

// CrashRule detects application crashes (fatal errors, panics, process
// exits). If out-of-memory evidence is present, the crash is treated as a
// consequence of OOM and this rule downgrades itself.
type CrashRule struct{}

// NewCrashRule returns a CrashRule.
func NewCrashRule() *CrashRule { return &CrashRule{} }

// ID implements diagnosis.DiagnosisRule.
func (*CrashRule) ID() string { return "application_crash" }

var crashKeywords = []string{
	"fatal", "panic", "uncaught exception", "process exited", "application crashed", "segmentation fault",
}

var crashRecommendations = []string{
	"Check application process status and restart reason",
	"Inspect crash dumps and core files",
	"Review recent code deployment or configuration change",
	"Check resource usage before the crash",
}

// Evaluate implements diagnosis.DiagnosisRule.
func (*CrashRule) Evaluate(ctx *diagnosis.DiagnosisContext) *model.Diagnosis {
	count, first, _ := diagnosis.CountKeywordGroups(ctx.ErrorGroups, crashKeywords)
	if count == 0 {
		return nil
	}

	conf := diagnosis.ScoreSupporting
	evidence := []model.Evidence{
		diagnosis.NewErrorPatternEvidence(first, "crash indicators detected", diagnosis.ScoreSupporting),
	}

	if count >= crashVolumeThreshold {
		conf += diagnosis.ScoreSupporting
		evidence = append(evidence,
			diagnosis.NewErrorPatternEvidence(first, "multiple crash indicators", diagnosis.ScoreSupporting))
	}

	// Stack trace evidence.
	if hasCrashStackTrace(ctx) {
		conf += diagnosis.ScoreSupporting
		evidence = append(evidence,
			diagnosis.NewStackTraceEvidence(first, "crash present in stack trace", diagnosis.ScoreSupporting))
	}

	// Contradiction: OOM explains the crash; prefer the OOM verdict.
	if oomPresent(ctx) {
		conf += diagnosis.ScoreContradict
		evidence = append(evidence,
			model.Evidence{
				Timestamp: first,
				Type:      model.EvidenceErrorPattern,
				Message:   "out-of-memory evidence present; crash is a likely consequence",
				Weight:    diagnosis.ScoreContradict,
			})
	}

	return &model.Diagnosis{
		RootCause:       "Application crash",
		Confidence:      conf,
		Severity:        model.SeverityHigh,
		Evidence:        evidence,
		Recommendations: crashRecommendations,
	}
}

// hasCrashStackTrace reports whether any crash group captured a stack trace
// in its examples.
func hasCrashStackTrace(ctx *diagnosis.DiagnosisContext) bool {
	for i := range ctx.ErrorGroups {
		g := &ctx.ErrorGroups[i]
		if !diagnosis.ContainsAny(g.Message, crashKeywords) {
			continue
		}
		for _, ex := range g.Examples {
			if ex.StackTrace != "" {
				return true
			}
		}
	}
	return false
}

// oomPresent reports whether OOM evidence exists in the groups.
func oomPresent(ctx *diagnosis.DiagnosisContext) bool {
	count, _, _ := diagnosis.CountKeywordGroups(ctx.ErrorGroups, oomKeywords)
	return count > 0
}
