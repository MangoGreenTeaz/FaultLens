// Package grouping clusters error events that share the same normalized
// message shape and assigns each cluster a stable SHA-256 fingerprint.
package grouping

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"time"

	"github.com/faultlens/faultlens/internal/model"
	"github.com/faultlens/faultlens/internal/normalize"
)

// defaultMaxExamples bounds how many raw examples are retained per group.
const defaultMaxExamples = 3

// ErrorGroup is a cluster of error events with an identical normalized
// message.
type ErrorGroup struct {
	Fingerprint string
	Message     string
	Count       int
	FirstSeen   time.Time
	LastSeen    time.Time
	Examples    []model.LogEvent
}

// Grouper accumulates error events into ErrorGroup clusters.
type Grouper struct {
	norm        *normalize.Normalizer
	groups      map[string]*ErrorGroup
	maxExamples int
}

// New returns a Grouper with default settings.
func New() *Grouper {
	return &Grouper{
		norm:        normalize.New(),
		groups:      make(map[string]*ErrorGroup),
		maxExamples: defaultMaxExamples,
	}
}

// Add folds an event into its cluster. Only ERROR and FATAL events are
// grouped; other levels are ignored.
func (g *Grouper) Add(ev *model.LogEvent) {
	if ev == nil {
		return
	}
	if ev.Level != model.LevelError && ev.Level != model.LevelFatal {
		return
	}

	norm := g.norm.Normalize(ev.Message)
	fp := Fingerprint(norm)

	group := g.groups[fp]
	if group == nil {
		group = &ErrorGroup{Fingerprint: fp, Message: norm}
		g.groups[fp] = group
	}

	group.Count++
	if group.FirstSeen.IsZero() || ev.Timestamp.Before(group.FirstSeen) {
		group.FirstSeen = ev.Timestamp
	}
	if ev.Timestamp.After(group.LastSeen) {
		group.LastSeen = ev.Timestamp
	}
	if len(group.Examples) < g.maxExamples {
		group.Examples = append(group.Examples, *ev)
	}
}

// Groups returns all clusters sorted by Count descending, then by FirstSeen
// ascending for stable output.
func (g *Grouper) Groups() []ErrorGroup {
	out := make([]ErrorGroup, 0, len(g.groups))
	for _, grp := range g.groups {
		out = append(out, *grp)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].FirstSeen.Before(out[j].FirstSeen)
	})
	return out
}

// TotalGroups returns the number of distinct clusters.
func (g *Grouper) TotalGroups() int { return len(g.groups) }

// Fingerprint returns a stable SHA-256 hex digest of a normalized message.
// It depends only on the message content, never on map iteration order.
func Fingerprint(normalized string) string {
	h := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(h[:])
}
