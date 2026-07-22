package sync_test

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"go-imapsync/internal/config"
	"go-imapsync/internal/identity"
	"go-imapsync/internal/imapclient"
	"go-imapsync/internal/secret"
	isync "go-imapsync/internal/sync"
	"go-imapsync/internal/testutil"
)

func testCfg(srv *testutil.Server, dry, justFolders bool) *config.Config {
	cfg := &config.Config{
		Host1: config.Side{
			Host:     srv.Addr,
			Port:     srv.Port,
			User:     srv.SrcUser,
			Password: secret.New(srv.SrcPass),
			SSL:      true,
		},
		Host2: config.Side{
			Host:     srv.Addr,
			Port:     srv.Port,
			User:     srv.DstUser,
			Password: secret.New(srv.DstPass),
			SSL:      true,
		},
		Dry:         dry,
		JustFolders: justFolders,
		UseHeader:   []string{"Message-Id"},
		Timeout:     10 * time.Second,
		InsecureTLS: false,
	}
	// Inject TLS via Dial options — Runner uses config only.
	// For tests we need TLSConfig on Dial; extend Runner or use InsecureTLS.
	// Memserver uses custom CA: use InsecureTLS for package-level sync tests
	// OR add TLSConfig to config — cleaner: set InsecureTLS true for unit sync
	// against self-signed... but Dial uses Side + InsecureTLS.
	cfg.InsecureTLS = true
	return cfg
}

// dialWithTLS is used only for verification after Runner runs.
// Runner itself uses InsecureTLS=true against self-signed in tests below.

func TestSyncTransfersAndDedups(t *testing.T) {
	msg := "Message-Id: <sync1@test>\r\nSubject: one\r\n\r\nbody one\r\n"
	srv := testutil.StartDualUserIMAP(t, []string{msg})
	// Custom CA: pass TLS via insecure skip for Runner (lab path under test).
	cfg := testCfg(srv, false, false)
	// InsecureTLS alone is not enough if ServerName fails — Dial uses
	// InsecureSkipVerify when InsecureTLS. Good.
	log := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
	runner := &isync.Runner{Cfg: cfg, Log: log}

	// Problem: Runner.Dial does not accept TLSConfig for custom RootCAs.
	// InsecureTLS sets InsecureSkipVerify — works with self-signed.
	stats, err := runner.Run(context.Background())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if stats.Transferred < 1 {
		t.Fatalf("transferred=%d, want >=1 (stats=%+v)", stats.Transferred, stats)
	}

	// Second run: all duplicates skipped.
	stats2, err := runner.Run(context.Background())
	if err != nil {
		t.Fatalf("second Run: %v", err)
	}
	if stats2.Transferred != 0 {
		t.Fatalf("second transferred=%d, want 0 (skipped=%d)", stats2.Transferred, stats2.Skipped)
	}
	if stats2.Skipped < 1 {
		t.Fatalf("second skipped=%d, want >=1", stats2.Skipped)
	}

	// Verify dest has the message key.
	ctx := context.Background()
	dst, err := imapclient.Dial(ctx, imapclient.Options{
		Label: "verify",
		Side: config.Side{
			Host: srv.Addr, Port: srv.Port, User: srv.DstUser,
			Password: secret.New(srv.DstPass), SSL: true,
		},
		Timeout: 5 * time.Second, TLSConfig: srv.TLS,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer dst.Close()
	if err := dst.Select(ctx, "INBOX", true); err != nil {
		t.Fatal(err)
	}
	metas, err := dst.FetchAllForIdentity(ctx, []string{"Message-Id"})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, m := range metas {
		k := identity.KeyFromHeaders(m.Headers, []string{"Message-Id"})
		if strings.Contains(k, "<sync1@test>") {
			found = true
		}
	}
	if !found {
		t.Fatal("destination missing transferred Message-Id")
	}
}

func TestSyncDryNoAppend(t *testing.T) {
	msg := "Message-Id: <dry@test>\r\n\r\nx\r\n"
	srv := testutil.StartDualUserIMAP(t, []string{msg})
	cfg := testCfg(srv, true, false)
	runner := &isync.Runner{Cfg: cfg, Log: slog.Default()}
	stats, err := runner.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if stats.Transferred < 1 {
		t.Fatalf("dry should count planned transfers, got %d", stats.Transferred)
	}

	ctx := context.Background()
	dst, err := imapclient.Dial(ctx, imapclient.Options{
		Label: "verify",
		Side: config.Side{
			Host: srv.Addr, Port: srv.Port, User: srv.DstUser,
			Password: secret.New(srv.DstPass), SSL: true,
		},
		Timeout: 5 * time.Second, TLSConfig: srv.TLS,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer dst.Close()
	if err := dst.Select(ctx, "INBOX", true); err != nil {
		t.Fatal(err)
	}
	metas, err := dst.FetchAllForIdentity(ctx, []string{"Message-Id"})
	if err != nil {
		t.Fatal(err)
	}
	if len(metas) != 0 {
		t.Fatalf("dry must not append, got %d messages", len(metas))
	}
}

func TestSyncJustFolders(t *testing.T) {
	msg := "Message-Id: <jf@test>\r\n\r\nx\r\n"
	srv := testutil.StartDualUserIMAP(t, []string{msg})
	cfg := testCfg(srv, false, true)
	runner := &isync.Runner{Cfg: cfg, Log: slog.Default()}
	stats, err := runner.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if stats.Transferred != 0 {
		t.Fatalf("justfolders transferred=%d, want 0", stats.Transferred)
	}
}
