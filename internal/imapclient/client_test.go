package imapclient_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"go-imapsync/internal/config"
	"go-imapsync/internal/identity"
	"go-imapsync/internal/imapclient"
	"go-imapsync/internal/secret"
	"go-imapsync/internal/testutil"
)

func TestDialAuthListFetch(t *testing.T) {
	msg := "Message-Id: <imap-client@test>\r\nSubject: hi\r\n\r\nbody\r\n"
	srv := testutil.StartDualUserIMAP(t, []string{msg})
	ctx := context.Background()

	src, err := imapclient.Dial(ctx, imapclient.Options{
		Label: "host1",
		Side: config.Side{
			Host:     srv.Addr,
			Port:     srv.Port,
			User:     srv.SrcUser,
			Password: secret.New(srv.SrcPass),
			SSL:      true,
		},
		Timeout:   5 * time.Second,
		TLSConfig: srv.TLS,
	})
	if err != nil {
		t.Fatalf("Dial src: %v", err)
	}
	defer src.Close()

	folders, err := src.ListFolders(ctx)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range folders {
		if f.Name == "INBOX" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("INBOX missing: %+v", folders)
	}

	if err := src.Select(ctx, "INBOX", true); err != nil {
		t.Fatal(err)
	}
	metas, err := src.FetchAllForIdentity(ctx, []string{"Message-Id"})
	if err != nil {
		t.Fatal(err)
	}
	if len(metas) != 1 {
		t.Fatalf("metas = %d, want 1", len(metas))
	}
	key := identity.KeyFromHeaders(metas[0].Headers, []string{"Message-Id"})
	if !strings.Contains(key, "<imap-client@test>") {
		t.Fatalf("key = %q", key)
	}

	full, err := src.FetchFull(ctx, metas[0].UID)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(full.Body), "body") {
		t.Fatalf("body = %q", full.Body)
	}
}

func TestDialBadPassword(t *testing.T) {
	srv := testutil.StartDualUserIMAP(t, nil)
	_, err := imapclient.Dial(context.Background(), imapclient.Options{
		Label: "host1",
		Side: config.Side{
			Host:     srv.Addr,
			Port:     srv.Port,
			User:     srv.SrcUser,
			Password: secret.New("wrong"),
			SSL:      true,
		},
		Timeout:   5 * time.Second,
		TLSConfig: srv.TLS,
	})
	if err == nil {
		t.Fatal("expected auth error")
	}
	if strings.Contains(err.Error(), "wrong") || strings.Contains(err.Error(), "srcpass") {
		t.Fatalf("error leaked password: %v", err)
	}
}

func TestAppendToDest(t *testing.T) {
	srv := testutil.StartDualUserIMAP(t, nil)
	ctx := context.Background()
	dst, err := imapclient.Dial(ctx, imapclient.Options{
		Label: "host2",
		Side: config.Side{
			Host:     srv.Addr,
			Port:     srv.Port,
			User:     srv.DstUser,
			Password: secret.New(srv.DstPass),
			SSL:      true,
		},
		Timeout:   5 * time.Second,
		TLSConfig: srv.TLS,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer dst.Close()

	body := []byte("Message-Id: <append@test>\r\n\r\nhello\r\n")
	if err := dst.Append(ctx, "INBOX", body, nil, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}
	if err := dst.Select(ctx, "INBOX", true); err != nil {
		t.Fatal(err)
	}
	metas, err := dst.FetchAllForIdentity(ctx, []string{"Message-Id"})
	if err != nil {
		t.Fatal(err)
	}
	if len(metas) != 1 {
		t.Fatalf("after append metas=%d", len(metas))
	}
}
