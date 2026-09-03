package config

import (
	"fmt"
	"strings"

	"github.com/MangoGreenTeaz/FaultLens/internal/model"
)

// Validate checks the configuration for invalid values and returns a
// descriptive error listing every problem found, or nil when valid.
func (c *Config) Validate() error {
	var problems []string

	if c.Anomaly.MinBaseline < 1 {
		problems = append(problems, "anomaly.min_baseline must be >= 1")
	}
	if c.Anomaly.ZScore <= 0 {
		problems = append(problems, "anomaly.z_score must be > 0")
	}
	if c.Anomaly.MinIncrease <= 0 {
		problems = append(problems, "anomaly.min_increase must be > 0")
	}
	if c.Anomaly.MinErrors < 0 {
		problems = append(problems, "anomaly.min_errors must be >= 0")
	}

	for id, r := range c.Rules {
		if r.Enabled != nil && !*r.Enabled {
			continue
		}
		if r.Threshold < 0 {
			problems = append(problems, fmt.Sprintf("rules.%s.threshold must be >= 0", id))
		}
	}

	seen := make(map[string]bool, len(c.CustomRules))
	for i, cr := range c.CustomRules {
		p := fmt.Sprintf("custom_rules[%d]", i)
		if cr.ID == "" {
			problems = append(problems, p+".id must not be empty")
		} else if seen[cr.ID] {
			problems = append(problems, p+".id "+cr.ID+" is duplicated")
		}
		seen[cr.ID] = true

		if cr.RootCause == "" {
			problems = append(problems, p+".root_cause must not be empty")
		}
		if len(cr.Keywords) == 0 {
			problems = append(problems, p+".keywords must contain at least one keyword")
		}
		if cr.StrongWeight < 0 || cr.StrongWeight > 1 {
			problems = append(problems, p+".strong_weight must be in [0, 1]")
		}
		if cr.SupportingWt < 0 || cr.SupportingWt > 1 {
			problems = append(problems, p+".supporting_weight must be in [0, 1]")
		}
		if !validSeverity(cr.Severity) {
			problems = append(problems, p+".severity must be one of low, medium, high, critical")
		}
	}

	if c.Output.Format != "" && !validOutputFormat(c.Output.Format) {
		problems = append(problems, "output.format must be one of terminal, json, markdown")
	}

	if len(problems) > 0 {
		return fmt.Errorf("invalid configuration:\n- %s", strings.Join(problems, "\n- "))
	}
	return nil
}

func validSeverity(s string) bool {
	switch model.Severity(s) {
	case model.SeverityLow, model.SeverityMedium, model.SeverityHigh, model.SeverityCritical:
		return true
	}
	return false
}

func validOutputFormat(s string) bool {
	switch s {
	case "terminal", "json", "markdown":
		return true
	}
	return false
}
