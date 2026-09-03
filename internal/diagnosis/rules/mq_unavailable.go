package rules

import (
	"github.com/MangoGreenTeaz/FaultLens/internal/diagnosis"
	"github.com/MangoGreenTeaz/FaultLens/internal/model"
)

// MQUnavailableRule detects a failing message queue (RabbitMQ / AMQP /
// Kafka client). It is an upstream failure: it outranks downstream 5xx.
type MQUnavailableRule struct{}

// NewMQUnavailableRule returns an MQUnavailableRule.
func NewMQUnavailableRule() *MQUnavailableRule { return &MQUnavailableRule{} }

// ID implements diagnosis.DiagnosisRule.
func (*MQUnavailableRule) ID() string { return "mq_unavailable" }

var mqStrongKeywords = []string{
	"rabbitmq", "amqp", "kafka client", "broker unavailable",
}

var mqWeakKeywords = []string{
	"connection refused", "connection timeout", "channel closed",
}

var mqRecommendations = []string{
	"Check message queue availability",
	"Check broker connection limit",
	"Check network connectivity between application and broker",
	"Inspect queue consumer lag",
}

// Evaluate implements diagnosis.DiagnosisRule.
func (*MQUnavailableRule) Evaluate(ctx *diagnosis.DiagnosisContext) *model.Diagnosis {
	strongCount, strongFirst, _ := diagnosis.CountKeywordGroups(ctx.ErrorGroups, mqStrongKeywords)
	if strongCount == 0 {
		return nil
	}

	conf := diagnosis.ScoreStrong
	evidence := []model.Evidence{
		diagnosis.NewErrorPatternEvidence(strongFirst, "message-queue errors detected", diagnosis.ScoreStrong),
	}

	if strongCount >= strongEvidenceThreshold {
		conf += diagnosis.ScoreSupporting
		evidence = append(evidence,
			diagnosis.NewErrorPatternEvidence(strongFirst, "large volume of queue errors", diagnosis.ScoreSupporting))
	}

	weakCount, weakFirst, _ := diagnosis.CountKeywordGroups(ctx.ErrorGroups, mqWeakKeywords)
	if weakCount > 0 {
		conf += diagnosis.ScoreSupporting
		evidence = append(evidence,
			diagnosis.NewErrorPatternEvidence(weakFirst, "connection failures detected", diagnosis.ScoreSupporting))
	}

	fiveCount, fiveFirst := diagnosis.Count5xxEvents(ctx)
	if fiveCount > 0 {
		conf += diagnosis.ScoreDownstream
		evidence = append(evidence,
			diagnosis.NewDownstreamEvidence(fiveFirst, "HTTP 5xx errors observed downstream", diagnosis.ScoreDownstream))
		if !strongFirst.IsZero() && !fiveFirst.IsZero() && strongFirst.Before(fiveFirst) {
			conf += diagnosis.ScoreTemporal
			evidence = append(evidence,
				diagnosis.NewTemporalEvidence(fiveFirst, "queue errors preceded HTTP 5xx spike", diagnosis.ScoreTemporal))
		}
	}

	return &model.Diagnosis{
		RootCause:       "Message queue unavailable",
		Confidence:      conf,
		Severity:        model.SeverityCritical,
		Evidence:        evidence,
		Recommendations: mqRecommendations,
	}
}
