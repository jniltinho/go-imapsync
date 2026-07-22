// Package report accumulates sync counters and formats the end-of-run summary.
package report

import (
	"fmt"
	"io"
	"time"
)

// Stats holds run counters.
type Stats struct {
	FoldersProcessed int
	FoldersCreated   int
	Transferred      int
	Skipped          int
	Failed           int
	Bytes            int64
	Started          time.Time
	Finished         time.Time
}

// New starts a stats clock.
func New() *Stats {
	return &Stats{Started: time.Now()}
}

// Finish marks completion time.
func (s *Stats) Finish() { s.Finished = time.Now() }

// OK reports whether the run had no hard message failures.
func (s *Stats) OK() bool { return s.Failed == 0 }

// WriteSummary writes an operator-friendly summary.
func (s *Stats) WriteSummary(w io.Writer) {
	if s.Finished.IsZero() {
		s.Finish()
	}
	dur := s.Finished.Sub(s.Started).Round(time.Millisecond)
	fmt.Fprintf(w, "---- go-imapsync summary ----\n")
	fmt.Fprintf(w, "Folders processed: %d  created: %d\n", s.FoldersProcessed, s.FoldersCreated)
	fmt.Fprintf(w, "Messages transferred: %d  skipped (already on host2): %d  failed: %d\n",
		s.Transferred, s.Skipped, s.Failed)
	fmt.Fprintf(w, "Bytes transferred: %d  duration: %s\n", s.Bytes, dur)
	if s.OK() {
		fmt.Fprintf(w, "The sync looks good: identified host1 messages in scope are present on host2 (or were skipped as duplicates).\n")
		fmt.Fprintf(w, "Detected 0 errors\n")
	} else {
		fmt.Fprintf(w, "The sync finished with %d error(s).\n", s.Failed)
	}
}
