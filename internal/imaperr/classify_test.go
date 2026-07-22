package imaperr

import (
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"testing"
)

func TestClassifyQuota(t *testing.T) {
	err := fmt.Errorf(`host2 APPEND write "INBOX": imap: NO [OVERQUOTA] Quota exceeded (mailbox for user is full) (0.001 + 0.000 secs).`)
	info := Classify(err)
	if info.Kind != KindQuota || !info.Fatal {
		t.Fatalf("got %+v", info)
	}
	if !strings.Contains(strings.ToLower(info.Message), "quota") {
		t.Fatalf("message: %s", info.Message)
	}
	if info.Hint == "" {
		t.Fatal("expected hint")
	}
}

func TestClassifyClosed(t *testing.T) {
	err := errors.New(`host2 APPEND write "INBOX": use of closed network connection`)
	info := Classify(err)
	if info.Kind != KindClosed {
		t.Fatalf("got %v", info.Kind)
	}
	info2 := Classify(net.ErrClosed)
	if info2.Kind != KindClosed {
		t.Fatalf("net.ErrClosed => %v", info2.Kind)
	}
	info3 := Classify(io.EOF)
	if info3.Kind != KindClosed {
		t.Fatalf("EOF => %v", info3.Kind)
	}
}

func TestClassifyAuth(t *testing.T) {
	info := Classify(errors.New(`host1 authentication failed for user "x": login failed`))
	if info.Kind != KindAuth {
		t.Fatalf("got %v", info.Kind)
	}
}

func TestClassifyTLS(t *testing.T) {
	info := Classify(errors.New(`host1 dial TLS: x509: certificate is valid for a, not b`))
	if info.Kind != KindTLS {
		t.Fatalf("got %v", info.Kind)
	}
}

func TestFormatAndDetail(t *testing.T) {
	err := fmt.Errorf(`host2 APPEND write "INBOX": imap: NO [OVERQUOTA] Quota exceeded (0.001 + 0.000 secs).`)
	s := Format("host2", "APPEND", err)
	if !strings.Contains(s, "quota") {
		t.Fatalf("format: %s", s)
	}
	d := Detail(err)
	if strings.Contains(d, "0.001") {
		t.Fatalf("detail should trim timing: %s", d)
	}
}
