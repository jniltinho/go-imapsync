// Package report accumulates sync counters and formats the end-of-run summary.
//
// Operators rely on WriteSummary for a clear success phrase or an error
// breakdown with “what to do next” hints after classified failures.
package report

import (
	"fmt"
	"io"
	"sort"
	"time"

	"go-imapsync/internal/imaperr"
)

// Stats holds run counters and optional abort/hint metadata for the summary.
type Stats struct {
	FoldersProcessed int
	FoldersCreated   int
	Transferred      int
	Skipped          int
	Failed           int
	Bytes            int64
	Started          time.Time
	Finished         time.Time

	// ByKind counts failures by classified kind (quota, connection_closed, …).
	ByKind map[string]int
	// Hints collects unique operator hints (capped).
	Hints []string
	// Aborted is set when a fatal condition stopped the run early.
	Aborted string
}

// New starts a stats clock with empty error maps.
func New() *Stats {
	return &Stats{
		Started: time.Now(),
		ByKind:  make(map[string]int),
	}
}

// Finish marks completion time used by duration reporting.
func (s *Stats) Finish() { s.Finished = time.Now() }

// OK reports whether the run had no failures and did not abort early.
func (s *Stats) OK() bool { return s.Failed == 0 && s.Aborted == "" }

// RecordError records a classified failure and optional operator hint.
func (s *Stats) RecordError(info imaperr.Info) {
	s.Failed++
	if s.ByKind == nil {
		s.ByKind = make(map[string]int)
	}
	s.ByKind[info.Kind.String()]++
	if info.Hint != "" {
		s.addHint(info.Hint)
	}
}

func (s *Stats) addHint(h string) {
	for _, x := range s.Hints {
		if x == h {
			return
		}
	}
	if len(s.Hints) >= 5 {
		return
	}
	s.Hints = append(s.Hints, h)
}

// WriteSummary writes an operator-friendly summary to w, including error
// breakdown and hints when the run was not fully successful.
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
		return
	}
	fmt.Fprintf(w, "The sync finished with %d error(s).\n", s.Failed)
	if s.Aborted != "" {
		fmt.Fprintf(w, "Stopped early: %s\n", s.Aborted)
	}
	if len(s.ByKind) > 0 {
		fmt.Fprintf(w, "Error breakdown:\n")
		keys := make([]string, 0, len(s.ByKind))
		for k := range s.ByKind {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(w, "  - %s: %d\n", k, s.ByKind[k])
		}
	}
	if len(s.Hints) > 0 {
		fmt.Fprintf(w, "What to do next:\n")
		for _, h := range s.Hints {
			fmt.Fprintf(w, "  • %s\n", h)
		}
	}
}
