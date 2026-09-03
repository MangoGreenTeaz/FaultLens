package rules

import (
	"github.com/MangoGreenTeaz/FaultLens/internal/diagnosis"
	"github.com/MangoGreenTeaz/FaultLens/internal/model"
)

// ConnectionPoolExhaustedRule detects connection pool starvation. Pool
// exhaustion is often a downstream symptom of an unavailable upstream
// (e.g. database); when strong database evidence exists this rule downgrades
// itself so the upstream cause wins.
type ConnectionPoolExhaustedRule struct{}

// NewConnectionPoolExhaustedRule returns a ConnectionPoolExhaustedRule.
func NewConnectionPoolExhaustedRule() *ConnectionPoolExhaustedRule {
	return &ConnectionPoolExhaustedRule{}
}

// ID implements diagnosis.DiagnosisRule.
func (*ConnectionPoolExhaustedRule) ID() string { return "connection_pool_exhausted" }

var poolKeywords = []string{
	"connection pool", "pool exhausted", "too many connections",
}

var poolRecommendations = []string{
	"Check connection pool size settings",
	"Inspect connection leak or unclosed connections",
	"Check upstream service health",
	"Check connection idle timeout configuration",
}

// Evaluate implements diagnosis.DiagnosisRule.
func (*ConnectionPoolExhaustedRule) Evaluate(ctx *diagnosis.DiagnosisContext) *model.Diagnosis {
	count, first, _ := diagnosis.CountKeywordGroups(ctx.ErrorGroups, poolKeywords)
	if count == 0 {
		return nil
	}

	conf := diagnosis.ScoreStrong
	evidence := []model.Evidence{
		diagnosis.NewErrorPatternEvidence(first, "connection pool exhaustion detected", diagnosis.ScoreStrong),
	}

	if count >= strongEvidenceThreshold {
		conf += diagnosis.ScoreSupporting
		evidence = append(evidence,
			diagnosis.NewErrorPatternEvidence(first, "large volume of pool errors", diagnosis.ScoreSupporting))
	}

	fiveCount, fiveFirst := diagnosis.Count5xxEvents(ctx)
	if fiveCount > 0 {
		conf += diagnosis.ScoreDownstream
		evidence = append(evidence,
			diagnosis.NewDownstreamEvidence(fiveFirst, "HTTP 5xx errors observed downstream", diagnosis.ScoreDownstream))
	}

	// Contradiction: an unavailable database explains pool exhaustion and
	// makes it a symptom, so the rule drops to symptom-level confidence.
	if databaseDown(ctx) {
		conf = diagnosis.ScoreSupporting
		evidence = append(evidence,
			model.Evidence{
				Timestamp: first,
				Type:      model.EvidenceErrorPattern,
				Message:   "database unavailable detected; pool exhaustion is a likely consequence",
				Weight:    diagnosis.ScoreContradict,
			})
	}

	return &model.Diagnosis{
		RootCause:       "Connection pool exhausted",
		Confidence:      conf,
		Severity:        model.SeverityHigh,
		Evidence:        evidence,
		Recommendations: poolRecommendations,
	}
}
