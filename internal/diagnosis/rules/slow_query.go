package rules

import (
	"github.com/faultlens/faultlens/internal/diagnosis"
	"github.com/faultlens/faultlens/internal/model"
)

// SlowQueryRule detects slow database queries. Slow queries are usually a
// symptom rather than a root cause, so this rule yields moderate confidence.
type SlowQueryRule struct{}

// NewSlowQueryRule returns a SlowQueryRule.
func NewSlowQueryRule() *SlowQueryRule { return &SlowQueryRule{} }

// ID implements diagnosis.DiagnosisRule.
func (*SlowQueryRule) ID() string { return "slow_query" }

var slowQueryKeywords = []string{
	"slow query", "query took", "latency exceeded", "slow sql",
}

var slowQueryRecommendations = []string{
	"Check query execution plans",
	"Review missing indexes",
	"Inspect database load and locks",
	"Check query parameterization and caching",
}

// Evaluate implements diagnosis.DiagnosisRule.
func (*SlowQueryRule) Evaluate(ctx *diagnosis.DiagnosisContext) *model.Diagnosis {
	count, first, _ := diagnosis.CountKeywordGroups(ctx.ErrorGroups, slowQueryKeywords)
	if count == 0 {
		return nil
	}

	// Symptom-type rule: starts from supporting weight, not strong.
	conf := diagnosis.ScoreSupporting
	evidence := []model.Evidence{
		diagnosis.NewErrorPatternEvidence(first, "slow query indicators detected", diagnosis.ScoreSupporting),
	}

	if count >= largeVolumeThreshold {
		conf += diagnosis.ScoreSupporting
		evidence = append(evidence,
			diagnosis.NewErrorPatternEvidence(first, "large volume of slow queries", diagnosis.ScoreSupporting))
	}

	fiveCount, fiveFirst := diagnosis.Count5xxEvents(ctx)
	if fiveCount > 0 {
		conf += diagnosis.ScoreDownstream
		evidence = append(evidence,
			diagnosis.NewDownstreamEvidence(fiveFirst, "HTTP 5xx errors observed downstream", diagnosis.ScoreDownstream))
	}

	return &model.Diagnosis{
		RootCause:       "Slow query",
		Confidence:      conf,
		Severity:        model.SeverityMedium,
		Evidence:        evidence,
		Recommendations: slowQueryRecommendations,
	}
}
