package rules

import (
	"github.com/faultlens/faultlens/internal/diagnosis"
	"github.com/faultlens/faultlens/internal/model"
)

// HTTPRule detects HTTP 5xx spikes. A 5xx spike is usually a symptom, not a
// root cause: when database or redis evidence is present, this rule
// downgrades itself so the upstream rule wins.
type HTTPRule struct{}

// NewHTTPRule returns an HTTPRule.
func NewHTTPRule() *HTTPRule { return &HTTPRule{} }

// ID implements diagnosis.DiagnosisRule.
func (*HTTPRule) ID() string { return "http_5xx_spike" }

var httpRecommendations = []string{
	"Check upstream service health",
	"Check recent deployment or configuration change",
	"Inspect load balancer and routing rules",
	"Check application error rate and latency",
}

// Evaluate implements diagnosis.DiagnosisRule.
func (*HTTPRule) Evaluate(ctx *diagnosis.DiagnosisContext) *model.Diagnosis {
	fiveCount, fiveFirst := diagnosis.Count5xxEvents(ctx.Events)
	if fiveCount == 0 {
		return nil
	}

	// A 5xx spike is a symptom, not a root cause: this rule deliberately
	// stays below the engine threshold on its own (supporting + anomaly =
	// 0.30 < 0.35), so pure 5xx logs yield "Insufficient evidence" instead
	// of a guessed upstream cause. The evidence still shows up for
	// explainability.
	conf := diagnosis.ScoreSupporting
	evidence := []model.Evidence{
		diagnosis.NewErrorPatternEvidence(fiveFirst, "HTTP 5xx errors detected", diagnosis.ScoreSupporting),
	}

	// Anomaly detector confirmation strengthens the spike signal.
	if len(ctx.Anomalies) > 0 {
		conf += diagnosis.ScoreAnomaly
		evidence = append(evidence,
			diagnosis.NewAnomalyEvidence(ctx.Anomalies[0].Bucket, "error rate anomaly detected", diagnosis.ScoreAnomaly))
	}

	// Contradiction: a clear upstream cause exists (database/redis down),
	// making the 5xx spike a symptom rather than the root cause.
	if databaseDown(ctx) || redisDown(ctx) {
		conf += diagnosis.ScoreContradict
		evidence = append(evidence,
			model.Evidence{
				Timestamp: fiveFirst,
				Type:      model.EvidenceErrorPattern,
				Message:   "upstream failure detected; 5xx spike is a downstream symptom",
				Weight:    diagnosis.ScoreContradict,
			})
	}

	return &model.Diagnosis{
		RootCause:       "HTTP 5xx spike",
		Confidence:      conf,
		Severity:        model.SeverityHigh,
		Evidence:        evidence,
		Recommendations: httpRecommendations,
	}
}

// databaseDown reports whether strong database evidence exists.
func databaseDown(ctx *diagnosis.DiagnosisContext) bool {
	count, _, _ := diagnosis.CountKeywordGroups(ctx.ErrorGroups, databaseStrongKeywords)
	return count >= strongEvidenceThreshold
}

// redisDown reports whether strong redis evidence exists.
func redisDown(ctx *diagnosis.DiagnosisContext) bool {
	count, _, _ := diagnosis.CountKeywordGroups(ctx.ErrorGroups, redisStrongKeywords)
	return count >= strongEvidenceThreshold
}
