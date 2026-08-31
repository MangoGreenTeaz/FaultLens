package rules

import (
	"github.com/faultlens/faultlens/internal/diagnosis"
	"github.com/faultlens/faultlens/internal/model"
)

// TimeoutRule detects connection/read/socket timeouts. Timeouts are usually a
// symptom of an upstream failure, so this rule yields moderate confidence.
type TimeoutRule struct{}

// NewTimeoutRule returns a TimeoutRule.
func NewTimeoutRule() *TimeoutRule { return &TimeoutRule{} }

// ID implements diagnosis.DiagnosisRule.
func (*TimeoutRule) ID() string { return "connection_timeout" }

var timeoutKeywords = []string{
	"connection timeout", "connect timeout", "read timeout", "i/o timeout", "socket timeout",
}

var timeoutRecommendations = []string{
	"Check network connectivity between services",
	"Check upstream service health and latency",
	"Check connection pool settings",
	"Check firewall and load balancer rules",
}

// Evaluate implements diagnosis.DiagnosisRule.
func (*TimeoutRule) Evaluate(ctx *diagnosis.DiagnosisContext) *model.Diagnosis {
	count, first, _ := diagnosis.CountKeywordGroups(ctx.ErrorGroups, timeoutKeywords)
	if count == 0 {
		return nil
	}

	conf := diagnosis.ScoreSupporting
	evidence := []model.Evidence{
		diagnosis.NewErrorPatternEvidence(first, "timeout errors detected", diagnosis.ScoreSupporting),
	}

	// Repeated timeouts across many events strengthen the signal.
	if count >= 20 {
		conf += diagnosis.ScoreSupporting
		evidence = append(evidence,
			diagnosis.NewErrorPatternEvidence(first, "large volume of timeout errors", diagnosis.ScoreSupporting))
	}

	fiveCount, fiveFirst := diagnosis.Count5xxEvents(ctx.Events)
	if fiveCount > 0 {
		conf += diagnosis.ScoreDownstream
		evidence = append(evidence,
			diagnosis.NewDownstreamEvidence(fiveFirst, "HTTP 5xx errors observed downstream", diagnosis.ScoreDownstream))
	}

	return &model.Diagnosis{
		RootCause:       "Connection timeout",
		Confidence:      conf,
		Severity:        model.SeverityMedium,
		Evidence:        evidence,
		Recommendations: timeoutRecommendations,
	}
}
