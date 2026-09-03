package rules

import (
	"github.com/MangoGreenTeaz/FaultLens/internal/diagnosis"
	"github.com/MangoGreenTeaz/FaultLens/internal/model"
)

// NetworkPartitionRule detects network-level failures (unreachable hosts,
// connection resets). It is an upstream failure that outranks downstream 5xx.
type NetworkPartitionRule struct{}

// NewNetworkPartitionRule returns a NetworkPartitionRule.
func NewNetworkPartitionRule() *NetworkPartitionRule { return &NetworkPartitionRule{} }

// ID implements diagnosis.DiagnosisRule.
func (*NetworkPartitionRule) ID() string { return "network_partition" }

var networkStrongKeywords = []string{
	"network unreachable", "host unreachable", "connection reset",
}

var networkRecommendations = []string{
	"Check network connectivity between hosts",
	"Check firewall and security group rules",
	"Inspect DNS resolution and routing",
	"Check load balancer health",
}

// Evaluate implements diagnosis.DiagnosisRule.
func (*NetworkPartitionRule) Evaluate(ctx *diagnosis.DiagnosisContext) *model.Diagnosis {
	strongCount, strongFirst, _ := diagnosis.CountKeywordGroups(ctx.ErrorGroups, networkStrongKeywords)
	if strongCount == 0 {
		return nil
	}

	conf := diagnosis.ScoreStrong
	evidence := []model.Evidence{
		diagnosis.NewErrorPatternEvidence(strongFirst, "network-level errors detected", diagnosis.ScoreStrong),
	}

	if strongCount >= strongEvidenceThreshold {
		conf += diagnosis.ScoreSupporting
		evidence = append(evidence,
			diagnosis.NewErrorPatternEvidence(strongFirst, "large volume of network errors", diagnosis.ScoreSupporting))
	}

	fiveCount, fiveFirst := diagnosis.Count5xxEvents(ctx)
	if fiveCount > 0 {
		conf += diagnosis.ScoreDownstream
		evidence = append(evidence,
			diagnosis.NewDownstreamEvidence(fiveFirst, "HTTP 5xx errors observed downstream", diagnosis.ScoreDownstream))
		if !strongFirst.IsZero() && !fiveFirst.IsZero() && strongFirst.Before(fiveFirst) {
			conf += diagnosis.ScoreTemporal
			evidence = append(evidence,
				diagnosis.NewTemporalEvidence(fiveFirst, "network errors preceded HTTP 5xx spike", diagnosis.ScoreTemporal))
		}
	}

	return &model.Diagnosis{
		RootCause:       "Network partition",
		Confidence:      conf,
		Severity:        model.SeverityCritical,
		Evidence:        evidence,
		Recommendations: networkRecommendations,
	}
}
