## Context

Application code at repo root is greenfield. Two read-only bases under `base/`:

| Path | Role |
|------|------|
| `base/imapsync` | **Behavior reference** — Perl imapsync 2.229 (~22k lines), FAQ, flags, examples |
| `base/go-getmail` | **Go implementation base** — [jniltinho/go-getmail](https://github.com/jniltinho/go-getmail): Cobra, go-imap/v2, TLS, secret redaction, Makefile |

Operators need a Go binary that performs one-way IMAP mailbox sync (host1 → host2) with imapsync-like CLI and safe defaults.

Primary validation path: build and run on the host, in Docker, and/or in a Vagrant Ubuntu 24.04 VM; optional live IMAP against `mail.linuxpro.com.br` with credentials supplied at runtime (not stored in the repo).

Reference principles (`base/imapsync/FAQ.d/FAQ.Principles.txt`): no config file, stateless, rsync-like, reliable, robust, safe defaults.

## Goals / Non-Goals

**Goals:**

- Ship a usable Go CLI that copies folders and messages host1 → host2 without duplicates.
- Match imapsync’s core identity model: headers (default Message-Id + Received) decide “already on host2”.
- Preserve message flags and reasonable dates (internal date from host1 when the server allows).
- Support SSL/TLS, dry-run, just-folders, structured logs, and a clear end-of-run summary.
- Make integration testing practical via Docker and/or Vagrant + live server hooks.
- Structure packages so later option parity can grow without rewriting the core loop.

**Non-Goals:**

- Full imapsync flag parity in this change (XOAUTH2, Gmail labels, proxyauth, NTLM, massive CSV UI, web UI).
- Two-way sync (offlineimap / mbsync territory).
- Destructive defaults (`--delete1`, `--delete2`, folder destruction) as first-class MVP path; may be stubbed/later.
- Embedding secrets, config files, or a daemon mode.
- Reusing go-getmail’s POP3, Maildir, TOML config, or UID-based oldmail state machine for product behavior.

## Decisions

### 0. Relationship to go-getmail (primary base for Go code)

**Decision:** Treat `base/go-getmail` as a **pattern and code-adapt source**, not a Go module dependency. Copy/adapt selected packages into this repo under our module path. Behavior and flags follow imapsync; engineering style follows go-getmail.

| go-getmail | Use in go-imapsync |
|------------|--------------------|
| `cmd/` Cobra, exit 0/1/2, signal context | **Adapt** — root sync command with imapsync flags (not `fetch`/`check` TOML workflow) |
| `internal/secret` (`String` redaction, LogValue) | **Adapt** almost as-is for passwords |
| `internal/retriever/imap.go` dial TLS, Login, Select, Fetch BODY[] | **Adapt** into dual-session `internal/imapclient` (source + dest); extend with LIST, CREATE, APPEND, flags, INTERNALDATE |
| `internal/retriever/pop3.go`, Maildir, state/oldmail, TOML config | **Do not use** for MVP product paths |
| `Makefile` (CGO_ENABLED=0, ldflags version, race tests, lint) | **Adapt** |
| Dedup `UIDVALIDITY:UID` | **Do not use** — imapsync uses header identity across servers |
| MIT license, no-secret-in-logs discipline | **Keep** |

**Rationale:** go-getmail already proves the IMAP TLS + Login + Fetch stack with the same library we want; reinventing dial/auth is waste. Sync orchestration and identity model are different products.

**Alternatives:** `go get` go-getmail and import packages — rejected: go-getmail’s IMAP type is a single-mailbox retriever tied to `fetch.Retriever`/`MessageID`, not a reusable dual-host sync client; coupling would fight both APIs. Better to adapt code locally.

### 1. Module and package layout

**Decision:** Mirror go-getmail’s `cmd/` + `internal/` style:

```text
cmd/                    # cobra tree (like go-getmail/cmd)
  root.go               # flags host1/2, dry, version; Execute + exit codes
  sync.go               # default / sync run (optional; may be root RunE)
internal/
  config/               # flag-derived options (not TOML)
  secret/               # adapted from go-getmail internal/secret
  imapclient/           # dual-role IMAP; evolve from go-getmail imap retriever
  identity/             # header keys (imapsync model)
  sync/                 # folder + message orchestration
  report/               # counters, summary, exit codes
  logx/                 # slog setup
testdata/
deploy/docker/
deploy/vagrant/
base/imapsync/          # read-only behavior reference
base/go-getmail/        # read-only Go reference (clone of github.com/jniltinho/go-getmail)
```

**Rationale:** Familiar layout for the same author/stack; clear seams for unit tests.

**Alternatives:** Flat `pkg/` public library first — deferred until a second consumer exists.

### 2. CLI style

**Decision:** **cobra + pflag**, same as go-getmail, but imapsync-compatible long options (`--host1`, `--user1`, `--password1`, …). Default command runs sync (operators expect `go-imapsync --host1 …` like Perl). Keep go-getmail’s exit code idea: 0 success, non-zero auth/runtime/usage. Optional later: `version` / `doctor` subcommands.

**Rationale:** Operators copy-paste imapsync examples; long-flag names MUST stay familiar. Cobra matches the proven go-getmail CLI.

**Alternatives:** Pure stdlib `flag` — fewer deps, worse multi-flag ergonomics.

### 3. IMAP library

**Decision:** Use **`github.com/emersion/go-imap/v2`** (and `go-message`, `go-sasl`) — same stack as go-getmail (`go.mod` pins v2 beta). Start from `base/go-getmail/internal/retriever/imap.go` patterns: `tls.Dialer`, `imapclient.New`, `Login`, `Select`, UID `Fetch` with body section, careful `Close` of body literals.

**Extend beyond go-getmail:** LIST (all folders), CREATE, APPEND (with flags + date), FETCH headers for identity building, optional STORE flags if needed, reconnect/re-SELECT for long syncs.

**Rationale:** Already battle-tested in go-getmail for mail.linuxpro-style IMAPS; avoids library thrash.

**Alternatives:**

- Raw net + hand-rolled IMAP — too costly/error-prone.
- Import go-getmail as library — API too specialized (see Decision 0).

### 4. Sync algorithm (MVP)

**Decision:** Sequential per-folder loop (host1 LIST → map folder names → ensure host2 folder → build host2 identity set → host1 FETCH candidates → APPEND missing). Stateless: identity sets rebuilt each run from IMAP state (no local DB required). Optional later: disk cache like imapsync `--usecache` (out of scope).

**Message identity:** Concatenate selected header fields (default `Message-Id` and `Received` lines) into a stable key; normalize whitespace; empty Message-Id falls back to a documented strategy (e.g. hash of headers + size, or skip with warning — implement “useheader Message-Id only” path early because FAQ recommends it for duplicates).

**Flags:** Copy system flags (`\Seen`, `\Answered`, `\Flagged`, `\Deleted`, `\Draft`) when host2 accepts them; log and continue on unsupported keywords.

**Dates:** Prefer setting APPEND internal date from host1 INTERNALDATE (`syncinternaldates` behavior on by default for MVP).

**Concurrency:** Single-threaded message transfer in MVP for correctness and simpler reconnect. Optional worker pool later behind a flag.

### 5. Reconnect and errors

**Decision:** On IMAP/IO errors, attempt reconnect with backoff for the failing side, re-SELECT folder, resume at next message. Cap retries; accumulate soft errors; hard-fail on auth failure or missing required flags. Exit non-zero if any hard error or if error count exceeds threshold (imapsync-style max errors).

**Ctrl-C:** First signal: attempt clean disconnect; second quick interrupt: exit immediately. Documented in help.

### 6. Security

**Decision:**

- Port go-getmail’s `secret.String` (redacted `String()`, `LogValue`, `MarshalText`) so passwords never appear in logs or `%v` dumps.
- Passwords via flags or env (`GOIMAPSYNC_PASSWORD1` / `GOIMAPSYNC_PASSWORD2`); optional later: `password_command` like go-getmail.
- TLS verify on by default; insecure skip only via explicit lab flag.
- No secrets in git, Docker images, or Vagrant scripts; `.env.example` placeholders only.
- Live tests gated by env `GOIMAPSYNC_LIVE=1` plus credential env vars.

### 7. Test strategy

**Decision:**

| Layer | How |
|-------|-----|
| Unit | identity keys, folder mapping, config validation, pure helpers |
| Fake IMAP | interface + mock client for sync loop |
| Integration (local) | Docker Compose with two Dovecot instances (optional) or single Dovecot + copy to second mailbox |
| Host integration | Vagrant Ubuntu 24.04 builds binary, runs unit tests |
| Live | Manual/CI optional: `--host1/--host2 mail.linuxpro.com.br` with operator credentials |

**Docker:** Multi-stage Dockerfile builds static binary; optional compose for self-contained smoke.

**Vagrant:** Box `ubuntu/jammy` or official Ubuntu 24.04; provision Go toolchain or copy prebuilt binary; run `go test ./...` and a dry-run smoke when live creds present.

### 8. Logging and report

**Decision:** `log/slog` text handler to stderr by default; optional `--logfile`. End summary includes: folders considered/created, messages transferred/skipped/errors, bytes, duration, exit code reason. Align wording loosely with imapsync (“sync looks good”) for operator familiarity without claiming byte-identical logs.

### 9. Phased parity (post-MVP, not blocking this change)

1. **Core (this change):** connect, folders, messages, identity, dry/justfolders, TLS, report.
2. **Selection:** `--folder`, `--folderrec`, `--exclude`, `--maxage`/`--minage`, `--search`.
3. **Mapping:** `--automap`, `--f1f2`, `--prefix1/2`, `--subfolder1/2`.
4. **Destructive:** `--delete2`, `--delete1`, expunge (explicit, well-tested).
5. **Auth extras:** XOAUTH2, authuser/proxyauth.
6. **Providers:** Gmail labels, Exchange quirks.

## Risks / Trade-offs

| Risk | Mitigation |
|------|------------|
| go-imap v2 API gaps vs Perl Mail::IMAPClient | Thin adapter; integration tests against real servers early |
| Header-based identity false positives/negatives | Default Message-Id+Received; document `--useheader`; unit tests with fixtures from FAQ |
| Server size/date quirks (imapsync FAQ) | Do not use size as identity; log size totals as informational only |
| Live server rate limits / lockouts | Dry-run first; throttle flag later; never run parallel destructive tests |
| Scope creep to full 22k-line parity | Strict MVP boundaries in tasks; later OpenSpec changes per phase |
| Credential leaks in CI logs | Redact secrets in logger; live tests manual by default |

## Migration Plan

N/A for end users (new tool). For developers:

1. Implement modules per `tasks.md`.
2. Validate with `go test`, Docker build, optional Vagrant.
3. Operator dry-run against `mail.linuxpro.com.br` when credentials available.
4. Iterate; archive this change when core scenarios pass; open follow-up changes for parity phases.

Rollback: delete binary / revert commits; no server-side schema to roll back (stateless).

## Open Questions

1. **Exact binary name:** `go-imapsync` vs `imapsync` (recommend `go-imapsync`, parallel to `go-getmail`).
2. **Live test accounts:** same server both sides (two mailboxes) vs host1 external → host2 linuxpro — operator will provide credentials later.
3. **Whether Docker Compose Dovecot is mandatory in MVP** or only Vagrant + live — recommend at least one automated IMAP path (mock or Dovecot) before relying solely on live.
4. **License:** recommend **MIT** like go-getmail (from-scratch Go; do not copy non-free UI assets from `base/imapsync`).
5. **How tightly to keep base/go-getmail updated:** shallow clone / submodule / manual refresh — for now a read-only checkout under `base/go-getmail` is enough.
