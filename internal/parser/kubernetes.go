package parser

import (
	"regexp"
	"time"

	"github.com/faultlens/faultlens/internal/model"
)

// Kubernetes container log format:
//
//	2026-08-31T14:32:01.123Z stdout F error: db down
var k8sRe = regexp.MustCompile(`^(\S+)\s+(stdout|stderr)\s+([FP])\s+(.*)$`)

// KubernetesParser parses kubelet container log lines.
type KubernetesParser struct{}

// NewKubernetesParser returns a KubernetesParser.
func NewKubernetesParser() *KubernetesParser { return &KubernetesParser{} }

// Name implements Parser.
func (*KubernetesParser) Name() string { return "kubernetes" }

// CanParse implements Parser.
func (*KubernetesParser) CanParse(line string) bool {
	return k8sRe.MatchString(line)
}

// Parse implements Parser.
func (*KubernetesParser) Parse(line string) []*model.LogEvent {
	m := k8sRe.FindStringSubmatch(line)
	if m == nil {
		return nil
	}

	ev := &model.LogEvent{
		Raw:     line,
		Message: m[4],
		Level:   model.LevelUnknown,
		Fields: map[string]string{
			"stream": m[2],
			"flags":  m[3],
		},
	}
	if t, err := time.Parse(time.RFC3339Nano, m[1]); err == nil {
		ev.Timestamp = t
	}
	return []*model.LogEvent{ev}
}

// Flush implements Parser. Kubernetes parsing is stateless.
func (*KubernetesParser) Flush() []*model.LogEvent { return nil }

// Issues implements Parser.
func (*KubernetesParser) Issues() int { return 0 }

// Reset implements Parser.
func (*KubernetesParser) Reset() {}
