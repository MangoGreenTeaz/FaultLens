package rules

import (
	"github.com/faultlens/faultlens/internal/diagnosis"
	"github.com/faultlens/faultlens/internal/model"
)

// DeadlockRule detects database/application deadlocks. It is an independent
// strong signal.
type DeadlockRule struct{}

// NewDeadlockRule returns a DeadlockRule.
func NewDeadlockRule() *DeadlockRule { return &DeadlockRule{} }

// ID implements diagnosis.DiagnosisRule.
func (*DeadlockRule) ID() string { return "deadlock" }

var deadlockKeywords = []string{
	"deadlock detected", "lock timeout", "deadlock found",
}

var deadlockRecommendations = []string{
	"Check database lock contention",
	"Inspect long-running transactions",
	"Review transaction isolation levels",
	"Check application locking order consistency",
}

// Evaluate implements diagnosis.DiagnosisRule.
func (*DeadlockRule) Evaluate(ctx *diagnosis.DiagnosisContext) *model.Diagnosis {
	count, first, _ := diagnosis.CountKeywordGroups(ctx.ErrorGroups, deadlockKeywords)
	if count == 0 {
		return nil
	}

	conf := diagnosis.ScoreStrong
	evidence := []model.Evidence{
		diagnosis.NewErrorPatternEvidence(first, "deadlock indicators detected", diagnosis.ScoreStrong),
	}

	if count >= strongEvidenceThreshold {
		conf += diagnosis.ScoreSupporting
		evidence = append(evidence,
			diagnosis.NewErrorPatternEvidence(first, "large volume of deadlock errors", diagnosis.ScoreSupporting))
	}

	return &model.Diagnosis{
		RootCause:       "Deadlock detected",
		Confidence:      conf,
		Severity:        model.SeverityHigh,
		Evidence:        evidence,
		Recommendations: deadlockRecommendations,
	}
}
