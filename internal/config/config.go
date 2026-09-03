// Package config loads, merges and validates FaultLens configuration.
//
// Configuration sources, lowest to highest priority:
//
//  1. Built-in defaults
//  2. Project config  (.faultlens.yaml)
//  3. User config     (~/.config/faultlens/config.yaml)
//  4. Explicit config (--config <file>)
//
// Higher-priority sources override lower ones field by field.
package config

// ProjectFileName is the project-level config file name.
const ProjectFileName = ".faultlens.yaml"

// AnomalyConfig tunes the anomaly detector.
type AnomalyConfig struct {
	MinBaseline int     `yaml:"min_baseline"`
	ZScore      float64 `yaml:"z_score"`
	MinIncrease float64 `yaml:"min_increase"`
	MinErrors   int     `yaml:"min_errors"`
}

// RuleConfig tunes one built-in diagnosis rule.
type RuleConfig struct {
	Enabled        *bool    `yaml:"enabled"`
	StrongKeywords []string `yaml:"strong_keywords"`
	WeakKeywords   []string `yaml:"weak_keywords"`
	Threshold      int      `yaml:"threshold"`
}

// CustomRuleConfig describes a user-defined diagnosis rule.
type CustomRuleConfig struct {
	ID               string   `yaml:"id"`
	RootCause        string   `yaml:"root_cause"`
	Severity         string   `yaml:"severity"`
	Keywords         []string `yaml:"keywords"`
	StrongWeight     float64  `yaml:"strong_weight"`
	SupportingKw     []string `yaml:"supporting_keywords"`
	SupportingWt     float64  `yaml:"supporting_weight"`
	EnableDownstream bool     `yaml:"enable_downstream"`
	Recommendations  []string `yaml:"recommendations"`
}

// OutputConfig tunes report rendering.
type OutputConfig struct {
	Format    string `yaml:"format"`
	MaxGroups int    `yaml:"max_groups"`
}

// Config is the complete resolved configuration.
type Config struct {
	Anomaly     AnomalyConfig         `yaml:"anomaly"`
	Rules       map[string]RuleConfig `yaml:"rules"`
	CustomRules []CustomRuleConfig    `yaml:"custom_rules"`
	Output      OutputConfig          `yaml:"output"`
	Parsers     map[string]bool       `yaml:"parsers"`
}

func boolPtr(b bool) *bool { return &b }

// Default returns the built-in default configuration. It must be safe to use
// without any config file present.
func Default() *Config {
	return &Config{
		Anomaly: AnomalyConfig{
			MinBaseline: 5,
			ZScore:      3.0,
			MinIncrease: 3.0,
			MinErrors:   10,
		},
		Rules: map[string]RuleConfig{
			"database_unavailable": {
				Enabled:        boolPtr(true),
				StrongKeywords: []string{"mysql", "postgres", "database", "sql", "jdbc"},
				WeakKeywords:   []string{"connection refused", "connection timeout"},
				Threshold:      10,
			},
			"redis_unavailable": {
				Enabled:        boolPtr(true),
				StrongKeywords: []string{"redis", "cache unavailable", "cache down"},
				WeakKeywords:   []string{"connection refused", "connection timeout", "timeout"},
				Threshold:      10,
			},
			"out_of_memory": {
				Enabled:        boolPtr(true),
				StrongKeywords: []string{"outofmemoryerror", "oomkilled", "out of memory", "memory limit", "java heap space"},
				Threshold:      1,
			},
			"connection_timeout": {
				Enabled:        boolPtr(true),
				StrongKeywords: []string{"connection timeout", "connect timeout", "read timeout", "i/o timeout", "socket timeout"},
				Threshold:      20,
			},
			"http_5xx_spike": {
				Enabled:   boolPtr(true),
				Threshold: 10,
			},
			"application_crash": {
				Enabled:        boolPtr(true),
				StrongKeywords: []string{"fatal", "panic", "uncaught exception", "process exited", "application crashed", "segmentation fault"},
				Threshold:      5,
			},
		},
		Output: OutputConfig{
			Format:    "terminal",
			MaxGroups: 5,
		},
		Parsers: map[string]bool{
			"plain": true,
			"json":  true,
			"java":  true,
			"nginx": true,
		},
	}
}
