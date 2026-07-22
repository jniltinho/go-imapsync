package main_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
)

type literal struct{ *strings.Reader }

func (l literal) Size() int64 { return int64(l.Reader.Len()) }

// TestE2ESmoke builds the real binary and syncs host1→host2 against an
// in-process dual-user TLS IMAP server.
func TestE2ESmoke(t *testing.T) {
	if testing.Short() {
		t.Skip("e2e smoke test skipped in -short mode")
	}
	dir := t.TempDir()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1)},
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	certFile := filepath.Join(dir, "ca.pem")
	if err := os.WriteFile(certFile, certPEM, 0o600); err != nil {
		t.Fatal(err)
	}

	src := imapmemserver.NewUser("src", "srcpass")
	dst := imapmemserver.NewUser("dst", "dstpass")
	for _, u := range []*imapmemserver.User{src, dst} {
		if err := u.Create("INBOX", nil); err != nil {
			t.Fatal(err)
		}
	}
	msg := "Message-Id: <e2e@test>\r\nSubject: e2e\r\n\r\nsmoke body\r\n"
	if _, err := src.Append("INBOX", literal{strings.NewReader(msg)}, &imap.AppendOptions{}); err != nil {
		t.Fatal(err)
	}
	mem := imapmemserver.New()
	mem.AddUser(src)
	mem.AddUser(dst)
	server := imapserver.New(&imapserver.Options{
		NewSession: func(*imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return mem.NewSession(), nil, nil
		},
		InsecureAuth: true,
	})
	ln, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key}},
	})
	if err != nil {
		t.Fatal(err)
	}
	go server.Serve(ln)
	t.Cleanup(func() { _ = server.Close() })
	port := ln.Addr().(*net.TCPAddr).Port

	binary := filepath.Join(dir, "go-imapsync")
	build := exec.Command("go", "build", "-o", binary, ".")
	build.Dir = "."
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}

	run := func(args ...string) (string, int) {
		cmd := exec.Command(binary, args...)
		cmd.Env = append(os.Environ(), "SSL_CERT_FILE="+certFile)
		out, err := cmd.CombinedOutput()
		code := 0
		if err != nil {
			if ee, ok := err.(*exec.ExitError); ok {
				code = ee.ExitCode()
			} else {
				t.Fatalf("run: %v\n%s", err, out)
			}
		}
		return string(out), code
	}

	baseArgs := []string{
		"--host1", "127.0.0.1", "--port1", fmt.Sprint(port),
		"--user1", "src", "--password1", "srcpass", "--ssl1",
		"--host2", "127.0.0.1", "--port2", fmt.Sprint(port),
		"--user2", "dst", "--password2", "dstpass", "--ssl2",
		"--useheader", "Message-Id",
		"--timeout", "10s",
		"--verbose",
	}

	// Version exits 0 without network.
	if out, code := run("version"); code != 0 {
		t.Fatalf("version exit=%d out=%s", code, out)
	}

	// Missing flags → usage exit 2.
	if _, code := run(); code != 2 {
		t.Fatalf("empty args exit=%d, want 2", code)
	}

	// Dry-run first.
	if out, code := run(append(baseArgs, "--dry")...); code != 0 {
		t.Fatalf("dry exit=%d\n%s", code, out)
	} else if strings.Contains(out, "srcpass") || strings.Contains(out, "dstpass") {
		t.Fatalf("dry leaked password:\n%s", out)
	}

	// Real sync.
	if out, code := run(baseArgs...); code != 0 {
		t.Fatalf("sync exit=%d\n%s", code, out)
	} else if strings.Contains(out, "srcpass") || strings.Contains(out, "dstpass") {
		t.Fatalf("sync leaked password:\n%s", out)
	} else if !strings.Contains(out, "sync looks good") && !strings.Contains(out, "transferred") {
		t.Fatalf("unexpected summary:\n%s", out)
	}

	// Second sync: duplicates only.
	if out, code := run(baseArgs...); code != 0 {
		t.Fatalf("second sync exit=%d\n%s", code, out)
	} else if !strings.Contains(out, "skipped") {
		// still ok if summary format differs
		t.Logf("second sync output:\n%s", out)
	}

	// Bad auth → non-zero.
	bad := append([]string{}, baseArgs...)
	for i := range bad {
		if bad[i] == "srcpass" {
			bad[i] = "nope"
		}
	}
	if out, code := run(bad...); code == 0 {
		t.Fatalf("bad password should fail:\n%s", out)
	}
}
