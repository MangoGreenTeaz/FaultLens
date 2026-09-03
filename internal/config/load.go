package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Load returns the fully merged configuration: built-in defaults, overlaid
// by the project config, the user config, and finally an explicit file.
// Missing project/user files are not errors; an explicit path must exist.
func Load(explicitPath string) (*Config, error) {
	cfg := Default()

	if err := mergeFile(cfg, ProjectFileName); err != nil {
		return nil, err
	}

	if dir, err := os.UserConfigDir(); err == nil {
		userPath := filepath.Join(dir, "faultlens", "config.yaml")
		if err := mergeFile(cfg, userPath); err != nil {
			return nil, err
		}
	}

	if explicitPath != "" {
		// An explicit path is a user error when missing; project and user
		// files are optional, an explicit file is not.
		if _, err := os.Stat(explicitPath); err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("config file %s does not exist", explicitPath)
			}
			return nil, err
		}
		if err := mergeFile(cfg, explicitPath); err != nil {
			return nil, err
		}
	}
	return cfg, nil
}

// mergeFile overlays one YAML file onto cfg. A missing file is ignored.
func mergeFile(cfg *Config, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read config %s: %w", path, err)
	}
	var overlay Config
	if err := yaml.Unmarshal(data, &overlay); err != nil {
		return fmt.Errorf("parse config %s: %w", path, err)
	}
	merge(cfg, &overlay)
	return nil
}

// merge overlays src onto dst. Zero values in src never overwrite dst; maps
// merge per key, and pointer booleans can explicitly set false.
func merge(dst, src *Config) {
	mergeAnomaly(&dst.Anomaly, &src.Anomaly)

	for k, v := range src.Rules {
		if existing, ok := dst.Rules[k]; ok {
			dst.Rules[k] = mergeRule(existing, v)
		} else {
			dst.Rules[k] = v
		}
	}

	if src.CustomRules != nil {
		dst.CustomRules = src.CustomRules
	}

	mergeOutput(&dst.Output, &src.Output)

	for k, v := range src.Parsers {
		dst.Parsers[k] = v
	}
}

func mergeAnomaly(dst, src *AnomalyConfig) {
	if src.MinBaseline != 0 {
		dst.MinBaseline = src.MinBaseline
	}
	if src.ZScore != 0 {
		dst.ZScore = src.ZScore
	}
	if src.MinIncrease != 0 {
		dst.MinIncrease = src.MinIncrease
	}
	if src.MinErrors != 0 {
		dst.MinErrors = src.MinErrors
	}
}

func mergeRule(dst, src RuleConfig) RuleConfig {
	if src.Enabled != nil {
		dst.Enabled = src.Enabled
	}
	if len(src.StrongKeywords) > 0 {
		dst.StrongKeywords = src.StrongKeywords
	}
	if len(src.WeakKeywords) > 0 {
		dst.WeakKeywords = src.WeakKeywords
	}
	if src.Threshold != 0 {
		dst.Threshold = src.Threshold
	}
	return dst
}

func mergeOutput(dst, src *OutputConfig) {
	if src.Format != "" {
		dst.Format = src.Format
	}
	if src.MaxGroups != 0 {
		dst.MaxGroups = src.MaxGroups
	}
}
