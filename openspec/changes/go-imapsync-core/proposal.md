## Why

Imapsync (Perl, ~22k lines, v2.229 in `base/imapsync`) is the de-facto tool for one-way, incremental IMAP mailbox migration, but it is hard to package, embed, or maintain as a modern single binary. This project needs a Go reimplementation (`go-imapsync`) that preserves the core “rsync for IMAP” behavior—stateless, restartable, duplicate-safe, safe defaults—while remaining easy to build, test, and ship.

We already have a solid Go mail client foundation in [jniltinho/go-getmail](https://github.com/jniltinho/go-getmail) (`base/go-getmail`): Cobra CLI, `emersion/go-imap/v2`, TLS dial, secret redaction, Makefile/release style. That codebase is the **implementation base** (patterns and adaptable code). Perl imapsync remains the **behavior reference**. We start with a useful core rather than full option parity with the Perl tool.

## What Changes

- Introduce a new Go module and CLI (`go-imapsync`) that connects to two IMAP servers and syncs folders and messages one way (host1 → host2).
- **Bootstrap from go-getmail patterns**: Cobra + exit-code conventions, `secret.String`-style password redaction, IMAP dial/login/SELECT/FETCH with `go-imap/v2`, slog, static `CGO_ENABLED=0` build Makefile — adapted into this repo (copy/adapt, not import as a runtime dependency of go-getmail).
- Implement core imapsync-compatible flags for credentials, TLS/SSL, folder listing/selection, dry-run, just-folders, message transfer with flags and internal dates, and **Message-Id/Received-based** duplicate detection (imapsync model — **not** go-getmail’s local `UIDVALIDITY:UID` oldmail state).
- Add dual-session IMAP work that go-getmail does not have: LIST hierarchy, CREATE folders on host2, APPEND with flags/dates, host2 identity set per folder.
- Add structured logging, exit codes, and summary stats comparable to imapsync’s “sync looks good” reporting.
- Provide test harnesses: unit tests, optional Docker image, and optional Vagrant Ubuntu 24.04 VM; live tests against `mail.linuxpro.com.br` using operator-supplied credentials (never committed).
- Keep `base/imapsync` and `base/go-getmail` as **read-only** references; do not modify those trees as part of product features.

## Capabilities

### New Capabilities

- `cli-sync`: Command-line interface for one-way IMAP sync (host1/user1/password1 ↔ host2/user2/password2), including connection options (port, SSL/TLS, dry-run, justfolders) and help/version.
- `imap-client`: IMAP connection, login, folder list/select/create, FETCH/APPEND, flags, reconnect and error handling for source and destination.
- `folder-sync`: Recursive folder hierarchy transfer, optional folder filters/mapping hooks for later parity, skip empty folders, INBOX safety.
- `message-sync`: Incremental message copy with identity by headers (default Message-Id + Received), preserve flags and dates, skip already-present messages, transfer progress and totals.
- `sync-report`: End-of-run summary (counts, sizes, errors, exit status) and optional logfile, aligned with imapsync operator expectations.
- `test-harness`: Docker and/or Vagrant Ubuntu 24.04 environments plus documented live test against `mail.linuxpro.com.br` (credentials via env/flags at runtime).

### Modified Capabilities

- (none — greenfield project; `openspec/specs/` has no existing capabilities)

## Impact

- **Codebase**: New Go module at repo root (`go.mod`, `cmd/`, `internal/`, tests). References: `base/imapsync` (behavior), `base/go-getmail` (Go patterns). Product code is new/adapted under repo root, not a submodule runtime link to go-getmail.
- **Dependencies**: Align with go-getmail where useful — `emersion/go-imap/v2`, `emersion/go-message`, `spf13/cobra`, `log/slog`; pin via `go.mod`. No POP3/Maildir/TOML requirement for MVP.
- **Ops/test**: Docker image and/or Vagrantfile for Ubuntu 24.04; integration tests require network access to IMAP servers; secrets only via env or CLI at test time (reuse redaction approach from go-getmail `internal/secret`).
- **Users**: Operators migrating or backing up mailboxes (including LinuxPro / `mail.linuxpro.com.br`) get a single static binary with familiar imapsync-style flags for the core path.
- **Non-goals (this change)**: Full imapsync option parity (XOAUTH2, Gmail labels, admin proxyauth, mass CSV UI, two-way sync, `--delete1`/`--delete2` as default path). Not a getmail replacement (no Maildir/POP3/oldmail). Those are later phases or other projects.
