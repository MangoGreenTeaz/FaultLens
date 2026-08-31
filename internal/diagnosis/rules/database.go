package rules

import (
	"github.com/faultlens/faultlens/internal/diagnosis"
	"github.com/faultlens/faultlens/internal/model"
)

// DatabaseRule detects an unavailable database (MySQL/Postgres/JDBC) by
// combining strong database-specific errors with downstream HTTP 5xx spikes.
type DatabaseRule struct{}

// NewDatabaseRule returns a DatabaseRule.
func NewDatabaseRule() *DatabaseRule { return &DatabaseRule{} }

// ID implements diagnosis.DiagnosisRule.
func (*DatabaseRule) ID() string { return "database_unavailable" }

// strongKeywords are database-specific signals.
var databaseStrongKeywords = []string{
	"mysql", "postgres", "database", "sql", "jdbc",
}

// weakKeywords only contribute when combined with strong database evidence.
var databaseWeakKeywords = []string{
	"connection refused", "connection timeout", "db unavailable", "database down",
}

var databaseRecommendations = []string{
	"Check MySQL availability",
	"Check database connection limit",
	"Check recent database restart",
	"Check network connectivity between application and database",
}

// Evaluate implements diagnosis.DiagnosisRule.
func (*DatabaseRule) Evaluate(ctx *diagnosis.DiagnosisContext) *model.Diagnosis {
	strongCount, strongFirst, _ := diagnosis.CountKeywordGroups(ctx.ErrorGroups, databaseStrongKeywords)
	if strongCount == 0 {
		// No database-specific error at all: not a database incident.
		return nil
	}

	conf := 0.0
	var evidence []model.Evidence

	// Strong evidence: database-specific error patterns.
	if strongCount >= 10 {
		conf += diagnosis.ScoreStrong
	} else {
		conf += diagnosis.ScoreSupporting
	}
	evidence = append(evidence,
		diagnosis.NewErrorPatternEvidence(strongFirst, "database-related errors detected", diagnosis.ScoreStrong))

	// Supporting evidence: generic connection failures around the same time.
	weakCount, weakFirst, _ := diagnosis.CountKeywordGroups(ctx.ErrorGroups, databaseWeakKeywords)
	if weakCount > 0 {
		conf += diagnosis.ScoreSupporting
		evidence = append(evidence,
			diagnosis.NewErrorPatternEvidence(weakFirst, "connection failures detected", diagnosis.ScoreSupporting))
	}

	// Downstream impact + temporal correlation: HTTP 5xx after the DB errors.
	fiveCount, fiveFirst := diagnosis.Count5xxEvents(ctx.Events)
	if fiveCount > 0 {
		conf += diagnosis.ScoreDownstream
		evidence = append(evidence,
			diagnosis.NewDownstreamEvidence(fiveFirst, "HTTP 5xx errors observed downstream", diagnosis.ScoreDownstream))
		if !strongFirst.IsZero() && !fiveFirst.IsZero() && strongFirst.Before(fiveFirst) {
			conf += diagnosis.ScoreTemporal
			evidence = append(evidence,
				diagnosis.NewTemporalEvidence(fiveFirst, "database errors preceded HTTP 5xx spike", diagnosis.ScoreTemporal))
		}
	}

	return &model.Diagnosis{
		RootCause:       "Database unavailable",
		Confidence:      conf,
		Severity:        model.SeverityCritical,
		Evidence:        evidence,
		Recommendations: databaseRecommendations,
	}
}
