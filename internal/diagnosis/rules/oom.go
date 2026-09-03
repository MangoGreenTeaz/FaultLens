package rules

import (
	"github.com/MangoGreenTeaz/FaultLens/internal/diagnosis"
	"github.com/MangoGreenTeaz/FaultLens/internal/model"
)

// OOMRule detects out-of-memory incidents. It has the highest severity and
// takes priority over generic application crash detection.
type OOMRule struct{}

// NewOOMRule returns an OOMRule.
func NewOOMRule() *OOMRule { return &OOMRule{} }

// ID implements diagnosis.DiagnosisRule.
func (*OOMRule) ID() string { return "out_of_memory" }

var oomKeywords = []string{
	"outofmemoryerror", "oomkilled", "out of memory", "memory limit", "java heap space",
}

var oomRecommendations = []string{
	"Check container memory limit",
	"Check application memory usage",
	"Inspect recent traffic increase",
	"Check heap configuration",
}

// Evaluate implements diagnosis.DiagnosisRule.
func (*OOMRule) Evaluate(ctx *diagnosis.DiagnosisContext) *model.Diagnosis {
	strongCount, first, _ := diagnosis.CountKeywordGroups(ctx.ErrorGroups, oomKeywords)
	if strongCount == 0 {
		return nil
	}

	conf := diagnosis.ScoreStrong
	evidence := []model.Evidence{
		diagnosis.NewErrorPatternEvidence(first, "out-of-memory errors detected", diagnosis.ScoreStrong),
	}

	// Stack trace evidence strengthens the verdict when the OOM appears
	// inside a captured Java stack trace.
	if hasOOMStackTrace(ctx) {
		conf += diagnosis.ScoreSupporting
		evidence = append(evidence,
			diagnosis.NewStackTraceEvidence(first, "out-of-memory error present in stack trace", diagnosis.ScoreSupporting))
	}

	if len(ctx.Anomalies) > 0 {
		conf += diagnosis.ScoreAnomaly
		evidence = append(evidence,
			diagnosis.NewAnomalyEvidence(ctx.Anomalies[0].Bucket, "error rate anomaly detected", diagnosis.ScoreAnomaly))
	}

	return &model.Diagnosis{
		RootCause:       "Out of memory",
		Confidence:      conf,
		Severity:        model.SeverityCritical,
		Evidence:        evidence,
		Recommendations: oomRecommendations,
	}
}

// hasOOMStackTrace reports whether any OOM error group captured a stack
// trace in its examples.
func hasOOMStackTrace(ctx *diagnosis.DiagnosisContext) bool {
	for i := range ctx.ErrorGroups {
		g := &ctx.ErrorGroups[i]
		if !diagnosis.ContainsAny(g.Message, oomKeywords) {
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
