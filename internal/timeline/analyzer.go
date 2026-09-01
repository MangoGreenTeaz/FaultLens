// Package timeline aggregates log events into fixed-width time buckets so
// that volume and error-rate changes over time can be analyzed.
package timeline

import (
	"sort"
	"time"

	"github.com/faultlens/faultlens/internal/model"
)

// DefaultInterval is the default bucket width (one minute).
const DefaultInterval = time.Minute

// Bucket counts log events that fell into one time window.
type Bucket struct {
	Start    time.Time `json:"start"`
	Total    int       `json:"total"`
	Errors   int       `json:"errors"`
	Warnings int       `json:"warnings"`
	Fatal    int       `json:"fatal"`
}

// Analyzer folds model.LogEvent values into Buckets.
type Analyzer struct {
	interval time.Duration
	buckets  map[int64]*Bucket
}

// New returns an Analyzer with the default one-minute interval.
func New() *Analyzer { return NewWithInterval(DefaultInterval) }

// NewWithInterval returns an Analyzer with a custom bucket width.
// Buckets are aligned to the interval boundary in UTC.
func NewWithInterval(interval time.Duration) *Analyzer {
	return &Analyzer{
		interval: interval,
		buckets:  make(map[int64]*Bucket),
	}
}

// Add folds an event into its time bucket. Events without a timestamp are
// ignored because they carry no temporal information.
func (a *Analyzer) Add(ev *model.LogEvent) {
	if ev == nil || ev.Timestamp.IsZero() {
		return
	}
	start := ev.Timestamp.Truncate(a.interval)
	key := start.Unix()

	b := a.buckets[key]
	if b == nil {
		b = &Bucket{Start: start}
		a.buckets[key] = b
	}

	b.Total++
	switch ev.Level {
	case model.LevelError:
		b.Errors++
	case model.LevelWarn:
		b.Warnings++
	case model.LevelFatal:
		b.Fatal++
	}
}

// Buckets returns all buckets sorted by start time ascending.
func (a *Analyzer) Buckets() []Bucket {
	out := make([]Bucket, 0, len(a.buckets))
	for _, b := range a.buckets {
		out = append(out, *b)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Start.Before(out[j].Start)
	})
	return out
}

// Interval returns the configured bucket width.
func (a *Analyzer) Interval() time.Duration { return a.interval }
