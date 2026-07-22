package secret

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"
	"time"
)

const sensitive = "hunter2-do-not-print"

func TestStringNeverPrints(t *testing.T) {
	s := New(sensitive)

	for name, got := range map[string]string{
		"String":    fmt.Sprint(s),
		"Sprintf v": fmt.Sprintf("value: %v", s),
		"Sprintf s": fmt.Sprintf("value: %s", s),
		"GoString":  fmt.Sprintf("%#v", s),
		"Sprintf+v": fmt.Sprintf("%+v", s),
	} {
		if strings.Contains(got, sensitive) {
			t.Errorf("%s leaked the secret: %q", name, got)
		}
	}

	if s.Reveal() != sensitive {
		t.Error("Reveal must return the real value")
	}
}

func TestStringJSONAndSlog(t *testing.T) {
	s := New(sensitive)

	data, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(data, []byte(sensitive)) {
		t.Errorf("JSON leaked the secret: %s", data)
	}

	var buf bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&buf, nil))
	logger.Info("test", "password", s)
	if strings.Contains(buf.String(), sensitive) {
		t.Errorf("slog leaked the secret: %s", buf.String())
	}
}

func TestStringUnmarshalText(t *testing.T) {
	var s String
	if err := s.UnmarshalText([]byte("abc")); err != nil {
		t.Fatal(err)
	}
	if s.Reveal() != "abc" {
		t.Errorf("Reveal = %q, want abc", s.Reveal())
	}
	if s.IsZero() {
		t.Error("IsZero must be false after UnmarshalText")
	}
}

func TestFromCommand(t *testing.T) {
	ctx := context.Background()

	t.Run("trims one trailing newline", func(t *testing.T) {
		s, err := FromCommand(ctx, []string{"printf", "secret\n"})
		if err != nil {
			t.Fatal(err)
		}
		if s.Reveal() != "secret" {
			t.Errorf("Reveal = %q, want secret", s.Reveal())
		}
	})

	t.Run("failure hides output", func(t *testing.T) {
		_, err := FromCommand(ctx, []string{"sh", "-c", "echo topsecret-output; exit 3"})
		if err == nil {
			t.Fatal("expected error")
		}
		if strings.Contains(err.Error(), "topsecret-output") {
			t.Errorf("error leaked command output: %v", err)
		}
	})

	t.Run("timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(ctx, 50*time.Millisecond)
		defer cancel()
		_, err := FromCommand(ctx, []string{"sleep", "10"})
		if err == nil {
			t.Fatal("expected timeout error")
		}
	})
}
