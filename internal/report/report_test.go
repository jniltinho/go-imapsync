package report

import (
	"bytes"
	"strings"
	"testing"

	"go-imapsync/internal/imaperr"
)

func TestWriteSummaryOK(t *testing.T) {
	s := New()
	s.Transferred = 2
	s.Skipped = 3
	s.FoldersProcessed = 1
	var buf bytes.Buffer
	s.WriteSummary(&buf)
	out := buf.String()
	if !strings.Contains(out, "transferred: 2") {
		t.Fatalf("output: %s", out)
	}
	if !strings.Contains(out, "sync looks good") {
		t.Fatalf("missing success phrase: %s", out)
	}
}

func TestWriteSummaryErrors(t *testing.T) {
	s := New()
	s.RecordError(imaperr.Info{
		Kind:    imaperr.KindQuota,
		Message: "quota exceeded",
		Hint:    "free space or raise quota, then re-run",
		Fatal:   true,
	})
	s.Aborted = "host2 mailbox quota exceeded"
	var buf bytes.Buffer
	s.WriteSummary(&buf)
	out := buf.String()
	if !strings.Contains(out, "1 error") {
		t.Fatalf("output: %s", out)
	}
	if !strings.Contains(out, "quota") {
		t.Fatalf("expected breakdown: %s", out)
	}
	if !strings.Contains(out, "What to do next") {
		t.Fatalf("expected hints: %s", out)
	}
}
