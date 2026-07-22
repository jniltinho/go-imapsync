package report

import (
	"bytes"
	"strings"
	"testing"
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
	s.Failed = 1
	var buf bytes.Buffer
	s.WriteSummary(&buf)
	if !strings.Contains(buf.String(), "1 error") {
		t.Fatalf("output: %s", buf.String())
	}
}
