package rules

import (
	"github.com/MangoGreenTeaz/FaultLens/internal/diagnosis"
	"github.com/MangoGreenTeaz/FaultLens/internal/model"
)

// CertificateExpiredRule detects TLS certificate failures. It is an
// independent strong signal.
type CertificateExpiredRule struct{}

// NewCertificateExpiredRule returns a CertificateExpiredRule.
func NewCertificateExpiredRule() *CertificateExpiredRule { return &CertificateExpiredRule{} }

// ID implements diagnosis.DiagnosisRule.
func (*CertificateExpiredRule) ID() string { return "certificate_expired" }

var certificateKeywords = []string{
	"certificate expired", "x509", "ssl handshake failed",
}

var certificateRecommendations = []string{
	"Check certificate expiry dates",
	"Renew the TLS certificate",
	"Verify certificate trust chain",
	"Check the system clock and time synchronization",
}

// Evaluate implements diagnosis.DiagnosisRule.
func (*CertificateExpiredRule) Evaluate(ctx *diagnosis.DiagnosisContext) *model.Diagnosis {
	count, first, _ := diagnosis.CountKeywordGroups(ctx.ErrorGroups, certificateKeywords)
	if count == 0 {
		return nil
	}

	conf := diagnosis.ScoreStrong
	evidence := []model.Evidence{
		diagnosis.NewErrorPatternEvidence(first, "certificate errors detected", diagnosis.ScoreStrong),
	}

	if count >= strongEvidenceThreshold {
		conf += diagnosis.ScoreSupporting
		evidence = append(evidence,
			diagnosis.NewErrorPatternEvidence(first, "large volume of certificate errors", diagnosis.ScoreSupporting))
	}

	return &model.Diagnosis{
		RootCause:       "Certificate expired",
		Confidence:      conf,
		Severity:        model.SeverityHigh,
		Evidence:        evidence,
		Recommendations: certificateRecommendations,
	}
}
