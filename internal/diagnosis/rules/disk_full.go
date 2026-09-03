package rules

import (
	"github.com/faultlens/faultlens/internal/diagnosis"
	"github.com/faultlens/faultlens/internal/model"
)

// DiskFullRule detects filesystem capacity exhaustion. It is an independent
// strong signal: no upstream cause is needed.
type DiskFullRule struct{}

// NewDiskFullRule returns a DiskFullRule.
func NewDiskFullRule() *DiskFullRule { return &DiskFullRule{} }

// ID implements diagnosis.DiagnosisRule.
func (*DiskFullRule) ID() string { return "disk_full" }

var diskKeywords = []string{
	"no space left on device", "disk full", "write error",
}

var diskRecommendations = []string{
	"Check filesystem capacity",
	"Clean temporary files",
	"Inspect log rotation settings",
	"Check the disk partition where the application writes",
}

// Evaluate implements diagnosis.DiagnosisRule.
func (*DiskFullRule) Evaluate(ctx *diagnosis.DiagnosisContext) *model.Diagnosis {
	count, first, _ := diagnosis.CountKeywordGroups(ctx.ErrorGroups, diskKeywords)
	if count == 0 {
		return nil
	}

	conf := diagnosis.ScoreStrong
	evidence := []model.Evidence{
		diagnosis.NewErrorPatternEvidence(first, "disk-full indicators detected", diagnosis.ScoreStrong),
	}

	if count >= strongEvidenceThreshold {
		conf += diagnosis.ScoreSupporting
		evidence = append(evidence,
			diagnosis.NewErrorPatternEvidence(first, "large volume of disk-full errors", diagnosis.ScoreSupporting))
	}

	if len(ctx.Anomalies) > 0 {
		conf += diagnosis.ScoreAnomaly
		evidence = append(evidence,
			diagnosis.NewAnomalyEvidence(ctx.Anomalies[0].Bucket, "error rate anomaly detected", diagnosis.ScoreAnomaly))
	}

	return &model.Diagnosis{
		RootCause:       "Disk full",
		Confidence:      conf,
		Severity:        model.SeverityCritical,
		Evidence:        evidence,
		Recommendations: diskRecommendations,
	}
}
