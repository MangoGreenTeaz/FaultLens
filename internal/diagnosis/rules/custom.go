package rules

import (
	"errors"
	"fmt"

	"github.com/MangoGreenTeaz/FaultLens/internal/config"
	"github.com/MangoGreenTeaz/FaultLens/internal/diagnosis"
	"github.com/MangoGreenTeaz/FaultLens/internal/model"
)

// Default weights applied when a custom rule does not set them explicitly.
const (
	defaultStrongWeight     = 0.40
	defaultSupportingWeight = 0.20
)

// CustomRule is a user-defined diagnosis rule loaded from configuration.
// It participates in the same Evidence → Confidence → Priority competition
// as the built-in rules.
type CustomRule struct {
	id               string
	RootCause        string
	Severity         model.Severity
	Keywords         []string
	StrongWeight     float64
	SupportingKw     []string
	SupportingWt     float64
	EnableDownstream bool
	Recommendations  []string
}

// ID implements diagnosis.DiagnosisRule.
func (r *CustomRule) ID() string { return r.id }

// Evaluate implements diagnosis.DiagnosisRule.
func (r *CustomRule) Evaluate(ctx *diagnosis.DiagnosisContext) *model.Diagnosis {
	count, first, _ := diagnosis.CountKeywordGroups(ctx.ErrorGroups, r.Keywords)
	if count == 0 {
		return nil
	}

	conf := r.StrongWeight
	evidence := []model.Evidence{
		diagnosis.NewErrorPatternEvidence(first, r.RootCause+" indicators detected", r.StrongWeight),
	}

	if len(r.SupportingKw) > 0 {
		if supCount, supFirst, _ := diagnosis.CountKeywordGroups(ctx.ErrorGroups, r.SupportingKw); supCount > 0 {
			conf += r.SupportingWt
			evidence = append(evidence,
				diagnosis.NewErrorPatternEvidence(supFirst, "supporting indicators detected", r.SupportingWt))
		}
	}

	if r.EnableDownstream {
		fiveCount, fiveFirst := diagnosis.Count5xxEvents(ctx)
		if fiveCount > 0 {
			conf += diagnosis.ScoreDownstream
			evidence = append(evidence,
				diagnosis.NewDownstreamEvidence(fiveFirst, "HTTP 5xx errors observed downstream", diagnosis.ScoreDownstream))
			if !first.IsZero() && !fiveFirst.IsZero() && first.Before(fiveFirst) {
				conf += diagnosis.ScoreTemporal
				evidence = append(evidence,
					diagnosis.NewTemporalEvidence(fiveFirst, r.RootCause+" errors preceded HTTP 5xx spike", diagnosis.ScoreTemporal))
			}
		}
	}

	return &model.Diagnosis{
		RootCause:       r.RootCause,
		Confidence:      conf,
		Severity:        r.Severity,
		Evidence:        evidence,
		Recommendations: r.Recommendations,
	}
}

// RegisterCustomRules registers every valid custom rule from configuration
// into the engine. Invalid rules are skipped and reported as warnings so they
// can never break the built-in rules; the number of successfully added rules
// is returned as added.
func RegisterCustomRules(e *diagnosis.Engine, cfgs []config.CustomRuleConfig) (added int, warnings []string) {
	for _, cr := range cfgs {
		rule, err := buildCustomRule(cr)
		if err != nil {
			warnings = append(warnings, fmt.Sprintf("skip custom rule %q: %v", cr.ID, err))
			continue
		}
		e.AddRule(rule)
		added++
	}
	return added, warnings
}

// buildCustomRule validates a config entry and applies default weights.
func buildCustomRule(cr config.CustomRuleConfig) (*CustomRule, error) {
	if cr.ID == "" {
		return nil, errors.New("id must not be empty")
	}
	if cr.RootCause == "" {
		return nil, errors.New("root_cause must not be empty")
	}
	if len(cr.Keywords) == 0 {
		return nil, errors.New("keywords must contain at least one keyword")
	}
	if cr.StrongWeight < 0 || cr.StrongWeight > 1 {
		return nil, fmt.Errorf("strong_weight must be in [0, 1], got %v", cr.StrongWeight)
	}
	if cr.SupportingWt < 0 || cr.SupportingWt > 1 {
		return nil, fmt.Errorf("supporting_weight must be in [0, 1], got %v", cr.SupportingWt)
	}

	sev := model.Severity(cr.Severity)
	switch sev {
	case model.SeverityLow, model.SeverityMedium, model.SeverityHigh, model.SeverityCritical:
	default:
		return nil, fmt.Errorf("severity must be one of low, medium, high, critical, got %q", cr.Severity)
	}

	sw := cr.StrongWeight
	if sw == 0 {
		sw = defaultStrongWeight
	}
	supw := cr.SupportingWt
	if supw == 0 {
		supw = defaultSupportingWeight
	}

	return &CustomRule{
		id:               cr.ID,
		RootCause:        cr.RootCause,
		Severity:         sev,
		Keywords:         cr.Keywords,
		StrongWeight:     sw,
		SupportingKw:     cr.SupportingKw,
		SupportingWt:     supw,
		EnableDownstream: cr.EnableDownstream,
		Recommendations:  cr.Recommendations,
	}, nil
}
