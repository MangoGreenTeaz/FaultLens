package rules

import (
	"github.com/faultlens/faultlens/internal/diagnosis"
	"github.com/faultlens/faultlens/internal/model"
)

// RedisRule detects an unavailable Redis / cache layer.
type RedisRule struct{}

// NewRedisRule returns a RedisRule.
func NewRedisRule() *RedisRule { return &RedisRule{} }

// ID implements diagnosis.DiagnosisRule.
func (*RedisRule) ID() string { return "redis_unavailable" }

var redisStrongKeywords = []string{
	"redis", "cache unavailable", "cache down",
}

var redisWeakKeywords = []string{
	"connection refused", "connection timeout", "timeout",
}

var redisRecommendations = []string{
	"Check Redis availability",
	"Check Redis connection limit",
	"Check network connectivity between application and Redis",
}

// Evaluate implements diagnosis.DiagnosisRule.
func (*RedisRule) Evaluate(ctx *diagnosis.DiagnosisContext) *model.Diagnosis {
	strongCount, strongFirst, _ := diagnosis.CountKeywordGroups(ctx.ErrorGroups, redisStrongKeywords)
	if strongCount == 0 {
		return nil
	}

	conf := 0.0
	var evidence []model.Evidence

	if strongCount >= strongEvidenceThreshold {
		conf += diagnosis.ScoreStrong
	} else {
		conf += diagnosis.ScoreSupporting
	}
	evidence = append(evidence,
		diagnosis.NewErrorPatternEvidence(strongFirst, "redis/cache errors detected", diagnosis.ScoreStrong))

	weakCount, weakFirst, _ := diagnosis.CountKeywordGroups(ctx.ErrorGroups, redisWeakKeywords)
	if weakCount > 0 {
		conf += diagnosis.ScoreSupporting
		evidence = append(evidence,
			diagnosis.NewErrorPatternEvidence(weakFirst, "connection failures detected", diagnosis.ScoreSupporting))
	}

	fiveCount, fiveFirst := diagnosis.Count5xxEvents(ctx.Events)
	if fiveCount > 0 {
		conf += diagnosis.ScoreDownstream
		evidence = append(evidence,
			diagnosis.NewDownstreamEvidence(fiveFirst, "HTTP 5xx errors observed downstream", diagnosis.ScoreDownstream))
		if !strongFirst.IsZero() && !fiveFirst.IsZero() && strongFirst.Before(fiveFirst) {
			conf += diagnosis.ScoreTemporal
			evidence = append(evidence,
				diagnosis.NewTemporalEvidence(fiveFirst, "redis errors preceded HTTP 5xx spike", diagnosis.ScoreTemporal))
		}
	}

	return &model.Diagnosis{
		RootCause:       "Redis unavailable",
		Confidence:      conf,
		Severity:        model.SeverityCritical,
		Evidence:        evidence,
		Recommendations: redisRecommendations,
	}
}
