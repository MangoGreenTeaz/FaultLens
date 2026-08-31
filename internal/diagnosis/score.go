package diagnosis

import (
	"regexp"
	"strings"
	"time"

	"github.com/faultlens/faultlens/internal/grouping"
	"github.com/faultlens/faultlens/internal/model"
)

// Confidence score components (section 21 of the spec). Every rule builds its
// confidence from these explainable building blocks.
const (
	ScoreStrong     = 0.40  // strong, directly matching evidence
	ScoreSupporting = 0.20  // additional supporting evidence
	ScoreTemporal   = 0.15  // temporal correlation with downstream impact
	ScoreDownstream = 0.15  // downstream impact observed
	ScoreAnomaly    = 0.10  // anomaly detector confirmation
	ScoreContradict = -0.20 // contradictory evidence (symptom treated as cause)
)

// ClampConfidence bounds a confidence value to [0, 1].
func ClampConfidence(f float64) float64 {
	if f < 0 {
		return 0
	}
	if f > 1 {
		return 1
	}
	return f
}

// ContainsAny reports whether s contains any of the given keywords
// (case-insensitive).
func ContainsAny(s string, keywords []string) bool {
	lower := strings.ToLower(s)
	for _, k := range keywords {
		if strings.Contains(lower, k) {
			return true
		}
	}
	return false
}

// CountKeywordGroups scans error groups whose normalized message contains any
// keyword and returns the total event count plus first/last seen times.
func CountKeywordGroups(groups []grouping.ErrorGroup, keywords []string) (count int, first, last time.Time) {
	for i := range groups {
		if !ContainsAny(groups[i].Message, keywords) {
			continue
		}
		g := &groups[i]
		count += g.Count
		if first.IsZero() || g.FirstSeen.Before(first) {
			first = g.FirstSeen
		}
		if g.LastSeen.After(last) {
			last = g.LastSeen
		}
	}
	return count, first, last
}

// fiveXXRe matches HTTP 5xx status codes in raw messages.
var fiveXXRe = regexp.MustCompile(`\b5\d\d\b`)

// Count5xxEvents scans error-level events for HTTP 5xx evidence: either the
// Nginx "status" field or a 5xx number in the raw message.
func Count5xxEvents(events []*model.LogEvent) (count int, first time.Time) {
	for _, e := range events {
		if e == nil {
			continue
		}
		hit := false
		if s, ok := e.Fields["status"]; ok && len(s) == 3 && s[0] == '5' {
			hit = true
		} else if fiveXXRe.MatchString(e.Message) {
			hit = true
		}
		if !hit {
			continue
		}
		count++
		if first.IsZero() || e.Timestamp.Before(first) {
			first = e.Timestamp
		}
	}
	return count, first
}

// Evidence helpers keep rule code concise and consistent.
func evAt(t time.Time, typ, msg string, weight float64) model.Evidence {
	return model.Evidence{Timestamp: t, Type: typ, Message: msg, Weight: weight}
}

// NewErrorPatternEvidence builds an ERROR_PATTERN evidence entry.
func NewErrorPatternEvidence(t time.Time, msg string, weight float64) model.Evidence {
	return evAt(t, model.EvidenceErrorPattern, msg, weight)
}

// NewDownstreamEvidence builds a DOWNSTREAM_IMPACT evidence entry.
func NewDownstreamEvidence(t time.Time, msg string, weight float64) model.Evidence {
	return evAt(t, model.EvidenceDownstreamImpact, msg, weight)
}

// NewTemporalEvidence builds a TIMELINE_CORRELATION evidence entry.
func NewTemporalEvidence(t time.Time, msg string, weight float64) model.Evidence {
	return evAt(t, model.EvidenceTimelineCorrelation, msg, weight)
}

// NewAnomalyEvidence builds an ANOMALY evidence entry.
func NewAnomalyEvidence(t time.Time, msg string, weight float64) model.Evidence {
	return evAt(t, model.EvidenceAnomaly, msg, weight)
}

// NewStackTraceEvidence builds a STACK_TRACE evidence entry.
func NewStackTraceEvidence(t time.Time, msg string, weight float64) model.Evidence {
	return evAt(t, model.EvidenceStackTrace, msg, weight)
}
