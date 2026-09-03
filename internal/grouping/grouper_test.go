package grouping

import (
	"testing"
	"time"

	"github.com/MangoGreenTeaz/FaultLens/internal/model"
)

func ev(level model.LogLevel, msg string, ts time.Time) *model.LogEvent {
	return &model.LogEvent{Level: level, Message: msg, Timestamp: ts}
}

func TestGrouperMergesDynamicVariants(t *testing.T) {
	g := New()
	base := time.Date(2026, 8, 31, 14, 32, 1, 0, time.UTC)
	g.Add(ev(model.LevelError, "Connection refused 10.0.0.1:3306", base))
	g.Add(ev(model.LevelError, "Connection refused 10.0.0.2:3306", base.Add(time.Second)))
	g.Add(ev(model.LevelError, "Connection refused 10.0.0.3:3306", base.Add(2*time.Second)))

	groups := g.Groups()
	if len(groups) != 1 {
		t.Fatalf("got %d groups, want 1 (dynamic variants must merge)", len(groups))
	}
	grp := groups[0]
	if grp.Count != 3 {
		t.Errorf("Count = %d, want 3", grp.Count)
	}
	if grp.Message != "Connection refused <IP>:<PORT>" {
		t.Errorf("Message = %q, want normalized shape", grp.Message)
	}
	if grp.Fingerprint != Fingerprint(grp.Message) {
		t.Errorf("Fingerprint mismatch")
	}
}

func TestGrouperSeparatesDistinctErrors(t *testing.T) {
	g := New()
	base := time.Date(2026, 8, 31, 14, 32, 1, 0, time.UTC)
	g.Add(ev(model.LevelError, "Connection refused", base))
	g.Add(ev(model.LevelError, "Connection timeout", base))
	g.Add(ev(model.LevelFatal, "OutOfMemoryError: Java heap space", base))

	groups := g.Groups()
	if len(groups) != 3 {
		t.Fatalf("got %d groups, want 3", len(groups))
	}
}

func TestGrouperSortsByCountDescending(t *testing.T) {
	g := New()
	base := time.Date(2026, 8, 31, 14, 32, 1, 0, time.UTC)
	g.Add(ev(model.LevelError, "rare error", base))
	for i := 0; i < 5; i++ {
		g.Add(ev(model.LevelError, "common error", base.Add(time.Duration(i)*time.Second)))
	}
	for i := 0; i < 2; i++ {
		g.Add(ev(model.LevelError, "medium error", base.Add(time.Duration(i)*time.Minute)))
	}

	groups := g.Groups()
	if len(groups) != 3 {
		t.Fatalf("got %d groups, want 3", len(groups))
	}
	if groups[0].Message != "common error" {
		t.Errorf("first group = %q, want the most frequent", groups[0].Message)
	}
	if groups[0].Count != 5 {
		t.Errorf("first group Count = %d, want 5", groups[0].Count)
	}
	if groups[2].Message != "rare error" {
		t.Errorf("last group = %q, want the least frequent", groups[2].Message)
	}
}

func TestGrouperTracksFirstAndLastSeen(t *testing.T) {
	g := New()
	base := time.Date(2026, 8, 31, 14, 32, 0, 0, time.UTC)
	g.Add(ev(model.LevelError, "boom", base))
	g.Add(ev(model.LevelError, "boom", base.Add(10*time.Second)))
	g.Add(ev(model.LevelError, "boom", base.Add(5*time.Second)))

	grp := g.Groups()[0]
	if !grp.FirstSeen.Equal(base) {
		t.Errorf("FirstSeen = %v, want %v", grp.FirstSeen, base)
	}
	if !grp.LastSeen.Equal(base.Add(10 * time.Second)) {
		t.Errorf("LastSeen = %v, want %v", grp.LastSeen, base.Add(10*time.Second))
	}
}

func TestGrouperLimitsExamples(t *testing.T) {
	g := New()
	base := time.Date(2026, 8, 31, 14, 32, 1, 0, time.UTC)
	for i := 0; i < 10; i++ {
		g.Add(ev(model.LevelError, "repeated failure", base.Add(time.Duration(i)*time.Second)))
	}
	grp := g.Groups()[0]
	if len(grp.Examples) != defaultMaxExamples {
		t.Errorf("Examples = %d, want %d", len(grp.Examples), defaultMaxExamples)
	}
}

func TestGrouperIgnoresNonErrorLevels(t *testing.T) {
	g := New()
	base := time.Date(2026, 8, 31, 14, 32, 1, 0, time.UTC)
	g.Add(ev(model.LevelInfo, "started", base))
	g.Add(ev(model.LevelWarn, "slow request", base))
	g.Add(ev(model.LevelDebug, "debug line", base))
	g.Add(ev(model.LevelTrace, "trace line", base))

	if g.TotalGroups() != 0 {
		t.Errorf("got %d groups, want 0 (INFO/WARN/DEBUG/TRACE are not grouped)", g.TotalGroups())
	}
}

func TestGrouperNilEventIgnored(t *testing.T) {
	g := New()
	g.Add(nil)
	if g.TotalGroups() != 0 {
		t.Errorf("got %d groups, want 0", g.TotalGroups())
	}
}

func TestFingerprintStable(t *testing.T) {
	a := Fingerprint("Connection refused <IP>:<PORT>")
	b := Fingerprint("Connection refused <IP>:<PORT>")
	if a != b {
		t.Errorf("Fingerprint not stable: %q != %q", a, b)
	}
	if len(a) != 64 {
		t.Errorf("Fingerprint length = %d, want 64 (SHA-256 hex)", len(a))
	}
}

func TestFingerprintDistinct(t *testing.T) {
	a := Fingerprint("Connection refused")
	b := Fingerprint("Connection timeout")
	if a == b {
		t.Error("distinct messages must have distinct fingerprints")
	}
}
