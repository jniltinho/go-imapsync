// Package testutil provides shared IMAP test helpers (in-memory TLS server).
//
// StartDualUserIMAP boots a dual-user IMAPS endpoint for unit/integration tests
// without contacting the network. Not for production use.
package testutil

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"io"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapserver"
	"github.com/emersion/go-imap/v2/imapserver/imapmemserver"
)

// Literal wraps a string reader with Size for APPEND.
type Literal struct{ *strings.Reader }

func (l Literal) Size() int64 { return int64(l.Reader.Len()) }

// NewLiteral returns an IMAP literal from s.
func NewLiteral(s string) Literal {
	return Literal{strings.NewReader(s)}
}

// Server is a TLS IMAP memserver with two users for host1→host2 tests.
type Server struct {
	Addr     string
	Port     int
	TLS      *tls.Config // client-side config trusting the test cert
	SrcUser  string
	SrcPass  string
	DstUser  string
	DstPass  string
	closeSrv io.Closer
}

// StartDualUserIMAP starts one TLS IMAP server with source and dest accounts.
// srcMsgs are appended to source INBOX before return.
func StartDualUserIMAP(t *testing.T, srcMsgs []string) *Server {
	t.Helper()

	srcUser, srcPass := "src", "srcpass"
	dstUser, dstPass := "dst", "dstpass"

	src := imapmemserver.NewUser(srcUser, srcPass)
	dst := imapmemserver.NewUser(dstUser, dstPass)
	if err := src.Create("INBOX", nil); err != nil {
		t.Fatal(err)
	}
	if err := dst.Create("INBOX", nil); err != nil {
		t.Fatal(err)
	}
	// Optional nested folder on source.
	if err := src.Create("Archive", nil); err != nil {
		t.Fatal(err)
	}
	for _, m := range srcMsgs {
		if _, err := src.Append("INBOX", NewLiteral(m), &imap.AppendOptions{
			Flags: []imap.Flag{imap.FlagSeen},
			Time:  time.Now().UTC(),
		}); err != nil {
			t.Fatal(err)
		}
	}

	mem := imapmemserver.New()
	mem.AddUser(src)
	mem.AddUser(dst)

	serverCfg, clientCfg := testTLS(t)
	srv := imapserver.New(&imapserver.Options{
		NewSession: func(*imapserver.Conn) (imapserver.Session, *imapserver.GreetingData, error) {
			return mem.NewSession(), nil, nil
		},
		InsecureAuth: true,
	})
	ln, err := tls.Listen("tcp", "127.0.0.1:0", serverCfg)
	if err != nil {
		t.Fatal(err)
	}
	go srv.Serve(ln)
	t.Cleanup(func() { _ = srv.Close() })

	port := ln.Addr().(*net.TCPAddr).Port
	return &Server{
		Addr:     "127.0.0.1",
		Port:     port,
		TLS:      clientCfg,
		SrcUser:  srcUser,
		SrcPass:  srcPass,
		DstUser:  dstUser,
		DstPass:  dstPass,
		closeSrv: srv,
	}
}

func testTLS(t *testing.T) (serverCfg, clientCfg *tls.Config) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1)},
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	cert := tls.Certificate{Certificate: [][]byte{der}, PrivateKey: key}
	serverCfg = &tls.Config{Certificates: []tls.Certificate{cert}}

	pool := x509.NewCertPool()
	pool.AddCert(mustParse(t, der))
	clientCfg = &tls.Config{RootCAs: pool, ServerName: "127.0.0.1"}
	return serverCfg, clientCfg
}

func mustParse(t *testing.T, der []byte) *x509.Certificate {
	t.Helper()
	c, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return c
}
