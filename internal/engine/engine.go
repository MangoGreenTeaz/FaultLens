// Package engine wires the whole analysis pipeline together: streaming input
// → parser → normalize → grouping → timeline → anomaly → diagnosis.
//
// It is the reusable core behind the CLI and stays independent of Cobra.
package engine

import (
	"io"
	"os"
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
	// Format is "auto", "plain", "json", "java", "nginx", "apache",
	// "python", "syslog", "docker" or "kubernetes".
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

// analyzer accumulates events from one or more inputs into a single result.
type analyzer struct {
	opts        Options
	grouper     *grouping.Grouper
	tl          *timeline.Analyzer
	sum         Summary
	fiveXXCount int
	fiveXXFirst time.Time
	detected    string // first auto-detected format across inputs
}

// newAnalyzer creates an empty analyzer for the given options.
func newAnalyzer(opts Options) *analyzer {
	return &analyzer{
		opts:    opts,
		grouper: grouping.New(),
		tl:      timeline.New(),
		sum:     Summary{Source: opts.Source, Format: opts.Format},
	}
}

// Run executes the full pipeline over a single stream and returns the
// aggregated result.
func Run(r io.Reader, opts Options) (*Result, error) {
	a := newAnalyzer(opts)
	if err := a.consume(r, opts.Source, opts.Format); err != nil {
		return nil, err
	}
	return a.result(), nil
}

// RunFiles executes the pipeline over a list of files, merging all events
// into one analysis. Each event's Source is the file it came from, so
// error-group examples stay traceable to their origin file.
func RunFiles(paths []string, opts Options) (*Result, error) {
	a := newAnalyzer(opts)
	for _, path := range paths {
		f, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		cerr := a.consume(f, path, opts.Format)
		f.Close()
		if cerr != nil {
			return nil, cerr
		}
	}
	return a.result(), nil
}

// consume streams one input into the shared aggregators. Each input gets a
// fresh parser, so auto format detection is per file.
func (a *analyzer) consume(r io.Reader, source, format string) error {
	rd := input.NewReader(r, source)
	p := newParser(format)

	// Stamp every produced event with its origin so error-group examples
	// stay traceable to the file they came from.
	setSource := func(evs []*model.LogEvent) {
		for _, ev := range evs {
			if ev.Source == "" {
				ev.Source = source
			}
		}
	}

	for rd.Scan() {
		evs := p.Parse(rd.Text())
		setSource(evs)
		a.consumeEvents(evs)
	}
	if err := rd.Err(); err != nil {
		return err
	}
	evs := p.Flush()
	setSource(evs)
	a.consumeEvents(evs)

	if ap, ok := p.(*parser.AutoParser); ok {
		if d := ap.Detected(); d != "" && a.detected == "" {
			a.detected = d
			a.sum.Format = d
		}
	}
	a.sum.ParsingWarnings += p.Issues()
	return nil
}

// consumeEvents folds parsed events into the shared counters and aggregators.
func (a *analyzer) consumeEvents(evs []*model.LogEvent) {
	for _, ev := range evs {
		if !inRange(ev.Timestamp, a.opts.From, a.opts.To) {
			continue
		}
		a.sum.Events++
		switch ev.Level {
		case model.LevelError:
			a.sum.Errors++
		case model.LevelWarn:
			a.sum.Warnings++
		case model.LevelFatal:
			a.sum.Fatal++
		}
		if !ev.Timestamp.IsZero() {
			if a.sum.FirstEvent.IsZero() || ev.Timestamp.Before(a.sum.FirstEvent) {
				a.sum.FirstEvent = ev.Timestamp
			}
			if ev.Timestamp.After(a.sum.LastEvent) {
				a.sum.LastEvent = ev.Timestamp
			}
		}
		if ev.Level == model.LevelError || ev.Level == model.LevelFatal {
			// HTTP 5xx is the only error detail the diagnosis rules need;
			// it is aggregated here so no full event slice is retained.
			if diagnosis.Is5xxEvent(ev) {
				a.fiveXXCount++
				if a.fiveXXFirst.IsZero() {
					a.fiveXXFirst = ev.Timestamp
				}
			}
		}
		a.grouper.Add(ev)
		a.tl.Add(ev)
	}
}

// result assembles the final Result from the shared aggregators.
func (a *analyzer) result() *Result {
	res := &Result{
		Summary:     a.sum,
		ErrorGroups: a.grouper.Groups(),
		Timeline:    a.tl.Buckets(),
	}

	cfg := a.opts.Config
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
		ErrorGroups: res.ErrorGroups,
		Timeline:    res.Timeline,
		Anomalies:   res.Anomalies,
		FiveXXCount: a.fiveXXCount,
		FiveXXFirst: a.fiveXXFirst,
	})
	return res
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
