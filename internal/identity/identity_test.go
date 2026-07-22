package identity

import (
	"strings"
	"testing"
)

func TestKeyFromHeadersMessageID(t *testing.T) {
	raw := []byte("From: a@b\r\nMessage-Id: <x@y>\r\nReceived: from z\r\n\r\nbody\r\n")
	k := KeyFromHeaders(raw, []string{"Message-Id"})
	if !strings.Contains(k, "<x@y>") {
		t.Fatalf("key = %q", k)
	}
	k2 := KeyFromHeaders(raw, []string{"Message-Id", "Received"})
	if k2 == k {
		t.Fatalf("expected Received to change key: %q", k2)
	}
}

func TestKeyDuplicateSame(t *testing.T) {
	a := []byte("Message-Id: <same@id>\r\nReceived: r1\r\n\r\n")
	b := []byte("Message-Id: <same@id>\r\nReceived: r1\r\n\r\n")
	if KeyFromHeaders(a, nil) != KeyFromHeaders(b, nil) {
		t.Fatal("keys should match")
	}
}

func TestKeyMissing(t *testing.T) {
	raw := []byte("Subject: no id\r\n\r\n")
	if KeyFromHeaders(raw, []string{"Message-Id"}) != "" {
		t.Fatal("expected empty key")
	}
}

func TestSplitHeadersBody(t *testing.T) {
	msg := []byte("A: 1\r\n\r\nhello")
	h, b := SplitHeadersBody(msg)
	if !strings.Contains(string(h), "A: 1") || string(b) != "hello" {
		t.Fatalf("h=%q b=%q", h, b)
	}
}
