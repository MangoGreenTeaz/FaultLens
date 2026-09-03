package engine

import (
	"fmt"
	"io"
	"runtime"
	"testing"
	"time"
)

// benchLineTpls are realistic mixed log lines; 3 of 5 are errors so the
// pipeline exercises grouping and diagnosis under load.
var benchLineTpls = []string{
	"2026-08-31 %s ERROR Connection refused 10.0.0.1:3306\n",
	"2026-08-31 %s INFO request handled\n",
	"2026-08-31 %s ERROR MySQL connection failed\n",
	"2026-08-31 %s WARN slow request took 3.2s\n",
	"2026-08-31 %s ERROR HTTP 500 Internal Server Error\n",
}

// syntheticLog produces a deterministic stream of realistic log lines
// without materializing the whole input in memory, so the engine's own
// allocations are what the benchmark measures.
type syntheticLog struct {
	remaining int64
	lines     int
}

// newSyntheticLog estimates the line count for the given input size.
func newSyntheticLog(sizeBytes int64) *syntheticLog {
	const avgLineLen = 75
	return &syntheticLog{remaining: sizeBytes / avgLineLen}
}

// Read implements io.Reader, emitting one line per call.
func (s *syntheticLog) Read(p []byte) (int, error) {
	if s.remaining <= 0 {
		return 0, io.EOF
	}
	sec := s.lines % 60
	minute := (s.lines / 60) % 60
	clock := fmt.Sprintf("14:%02d:%02d", minute, sec)
	line := fmt.Sprintf(benchLineTpls[s.lines%len(benchLineTpls)], clock)
	n := copy(p, line)
	s.remaining--
	s.lines++
	return n, nil
}

// BenchmarkRun128MB measures throughput and allocations over a 128 MiB
// synthetic log.
func BenchmarkRun128MB(b *testing.B) {
	const size = int64(128) * 1024 * 1024
	for i := 0; i < b.N; i++ {
		r := newSyntheticLog(size)
		b.SetBytes(size)
		b.ReportAllocs()
		if _, err := Run(r, Options{}); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRunSmall measures the per-run cost on a small input.
func BenchmarkRunSmall(b *testing.B) {
	const size = int64(1) * 1024 * 1024
	for i := 0; i < b.N; i++ {
		r := newSyntheticLog(size)
		b.SetBytes(size)
		b.ReportAllocs()
		if _, err := Run(r, Options{}); err != nil {
			b.Fatal(err)
		}
	}
}

// TestProcesses500MB verifies the V2 performance acceptance criterion: a
// 500 MiB log processes successfully on a normal machine. It records
// throughput and memory so results stay repeatable. Skipped under -short
// (CI); run locally without -short for the full check.
func TestProcesses500MB(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping 500MB check in short mode")
	}
	const size = int64(500) * 1024 * 1024

	var m0, m1 runtime.MemStats
	runtime.ReadMemStats(&m0)
	start := time.Now()

	res, err := Run(newSyntheticLog(size), Options{})
	elapsed := time.Since(start)
	runtime.ReadMemStats(&m1)

	if err != nil {
		t.Fatal(err)
	}
	if res.Summary.Events == 0 {
		t.Error("no events produced")
	}

	allocMB := (m1.TotalAlloc - m0.TotalAlloc) / (1024 * 1024)
	t.Logf("500MB: events=%d elapsed=%s throughput=%.1fMB/s totalAlloc=%dMB heapInuse=%dMB",
		res.Summary.Events, elapsed.Round(time.Millisecond),
		float64(size)/elapsed.Seconds()/(1024*1024), allocMB, m1.HeapInuse/(1024*1024))
}
