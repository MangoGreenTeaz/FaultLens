package model

import "time"

// Severity describes how critical a diagnosis is.
type Severity string

// Supported severities.
const (
	SeverityLow      Severity = "low"
	SeverityMedium   Severity = "medium"
	SeverityHigh     Severity = "high"
	SeverityCritical Severity = "critical"
)

// Evidence types used across all diagnosis rules.
const (
	EvidenceErrorPattern        = "ERROR_PATTERN"
	EvidenceAnomaly             = "ANOMALY"
	EvidenceTimelineCorrelation = "TIMELINE_CORRELATION"
	EvidenceDownstreamImpact    = "DOWNSTREAM_IMPACT"
	EvidenceStackTrace          = "STACK_TRACE"
)

// Evidence is one verifiable fact that contributed to a diagnosis.
type Evidence struct {
	Timestamp time.Time
	Type      string
	Message   string
	Weight    float64
}

// Diagnosis is the final root-cause verdict produced by the engine.
//
// Every non-trivial diagnosis must be backed by Evidence; Confidence is
// derived from the evidence score, never hard-coded. If the evidence is too
// weak, RootCause is "Insufficient evidence".
type Diagnosis struct {
	RootCause       string
	Confidence      float64
	Severity        Severity
	Evidence        []Evidence
	Recommendations []string
}
