// Package engine wires the whole analysis pipeline together: streaming input
// → parser → normalize → grouping → timeline → anomaly → diagnosis.
//
// It is the reusable core behind the CLI and stays independent of Cobra.
package engine

import (
	"io"
	"time"

	"github.com/faultlens/faultlens/internal/anomaly"
	"github.com/faultlens/faultlens/internal/config"
	"github.com/faultlens/faultlens/internal/diagnosis"
	"github.com/faultlens/faultlens/internal/diagnosis/rules"
	"github.com/faultlens/faultlens/internal/grouping"
	"github.com/faultlens/faultlens/internal/input"
	"github.com/faultlens/faultlens/internal/model"
	"github.com/faultlens/faultlens/internal/parser"
	"github.com/faultlens/faultlens/internal/timeline"
)

// Options controls one analysis run.
type Options struct {
	// Format is "auto", "plain", "json", "java" or "nginx".
	Format string
	// From/To filter events by timestamp (inclusive). Zero values disable
	// the corresponding bound.
	From time.Time
	To   time.Time
	// Source is a human-readable input description (file path or "stdin").
	Source string
	// Config carries the resolved configuration; nil means defaults.
	Config *config.Config
}

// Summary holds high-level counters for the whole input.
type Summary struct {
	Events          int       `json:"events"`
	Errors          int       `json:"errors"`
	Warnings        int       `json:"warnings"`
	Fatal           int       `json:"fatal"`
	FirstEvent      time.Time `json:"first_event"`
	LastEvent       time.Time `json:"last_event"`
	Format          string    `json:"format"`
	Source          string    `json:"source"`
	ParsingWarnings int       `json:"parsing_warnings"`
}

// Result is the complete output of one analysis run.
type Result struct {
	Summary     Summary               `json:"summary"`
	ErrorGroups []grouping.ErrorGroup `json:"error_groups"`
	Timeline    []timeline.Bucket     `json:"timeline"`
	Anomalies   []anomaly.Detection   `json:"anomalies"`
	Diagnosis   *model.Diagnosis      `json:"diagnosis"`
	// ConfigWarnings lists non-fatal configuration problems (e.g. invalid
	// custom rules that were skipped).
	ConfigWarnings []string `json:"config_warnings,omitempty"`
}

// Run executes the full pipeline over r and returns the aggregated result.
func Run(r io.Reader, opts Options) (*Result, error) {
	rd := input.NewReader(r, opts.Source)
	p := newParser(opts.Format)

	grouper := grouping.New()
	tl := timeline.New()
	var events []*model.LogEvent
	sum := Summary{Source: opts.Source, Format: opts.Format}

	consume := func(evs []*model.LogEvent) {
		for _, ev := range evs {
			if !inRange(ev.Timestamp, opts.From, opts.To) {
				continue
			}
			sum.Events++
			switch ev.Level {
			case model.LevelError:
				sum.Errors++
			case model.LevelWarn:
				sum.Warnings++
			case model.LevelFatal:
				sum.Fatal++
			}
			if !ev.Timestamp.IsZero() {
				if sum.FirstEvent.IsZero() || ev.Timestamp.Before(sum.FirstEvent) {
					sum.FirstEvent = ev.Timestamp
				}
				if ev.Timestamp.After(sum.LastEvent) {
					sum.LastEvent = ev.Timestamp
				}
			}
			if ev.Level == model.LevelError || ev.Level == model.LevelFatal {
				events = append(events, ev)
			}
			grouper.Add(ev)
			tl.Add(ev)
		}
	}

	for rd.Scan() {
		consume(p.Parse(rd.Text()))
	}
	if err := rd.Err(); err != nil {
		return nil, err
	}
	consume(p.Flush())

	if ap, ok := p.(*parser.AutoParser); ok {
		if d := ap.Detected(); d != "" {
			sum.Format = d
		}
	}
	sum.ParsingWarnings = p.Issues()

	res := &Result{
		Summary:     sum,
		ErrorGroups: grouper.Groups(),
		Timeline:    tl.Buckets(),
	}

	cfg := opts.Config
	if cfg == nil {
		cfg = config.Default()
	}
	res.Anomalies = anomaly.NewWithConfig(anomaly.Config{
		MinBaseline: cfg.Anomaly.MinBaseline,
		ZScore:      cfg.Anomaly.ZScore,
		MinIncrease: cfg.Anomaly.MinIncrease,
		MinErrors:   cfg.Anomaly.MinErrors,
	}).Detect(res.Timeline)

	eng := diagnosis.NewEngine()
	rules.RegisterDefaultRules(eng)
	if _, warnings := rules.RegisterCustomRules(eng, cfg.CustomRules); len(warnings) > 0 {
		res.ConfigWarnings = warnings
	}
	res.Diagnosis = eng.Diagnose(&diagnosis.DiagnosisContext{
		Events:      events,
		ErrorGroups: res.ErrorGroups,
		Timeline:    res.Timeline,
		Anomalies:   res.Anomalies,
	})
	return res, nil
}

// newParser maps a format flag to a concrete parser.
func newParser(format string) parser.Parser {
	switch format {
	case "plain":
		return parser.NewPlainTextParser()
	case "json":
		return parser.NewJSONParser()
	case "java":
		return parser.NewJavaParser()
	case "nginx":
		return parser.NewNginxParser()
	case "apache":
		return parser.NewApacheParser()
	case "python":
		return parser.NewPythonParser()
	case "syslog":
		return parser.NewSyslogParser()
	case "docker":
		return parser.NewDockerJSONParser()
	case "kubernetes":
		return parser.NewKubernetesParser()
	default:
		return parser.NewAutoParser()
	}
}

// inRange reports whether a timestamp falls inside the requested window.
// Events without a timestamp are only kept when no filter is active.
func inRange(t, from, to time.Time) bool {
	if !from.IsZero() && (t.IsZero() || t.Before(from)) {
		return false
	}
	if !to.IsZero() && (t.IsZero() || t.After(to)) {
		return false
	}
	return true
}
