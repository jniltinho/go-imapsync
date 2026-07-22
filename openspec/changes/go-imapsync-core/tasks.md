## 1. Project scaffolding (aligned with go-getmail)

- [ ] 1.1 Initialize Go module at repo root (`go.mod`); prefer path `github.com/jniltinho/go-imapsync` and Go version ≥ go-getmail (1.25+)
- [ ] 1.2 Create package layout: `cmd/`, `internal/{config,secret,imapclient,identity,sync,report,logx}` (mirror `base/go-getmail`)
- [ ] 1.3 Add README skeleton (build, flags overview, refs to `base/imapsync` + `base/go-getmail` / https://github.com/jniltinho/go-getmail, no secrets)
- [ ] 1.4 Add MIT LICENSE (like go-getmail) and `.gitignore` (binaries, `.env`, `.vagrant`, logs, ignore mutating `base/`)
- [ ] 1.5 Add `Makefile` adapted from `base/go-getmail/Makefile`: `build` (CGO_ENABLED=0, ldflags version), `test` (`go test -race ./...`), `lint`, `docker-build`

## 2. Port reusable pieces from go-getmail

- [ ] 2.1 Adapt `base/go-getmail/internal/secret` → `internal/secret` (redacted String, LogValue, tests)
- [ ] 2.2 Study `base/go-getmail/internal/retriever/imap.go` and extract dual-host client seams (connect/login/select/fetch body) into `internal/imapclient` stubs
- [ ] 2.3 Adapt Cobra root Execute pattern from `base/go-getmail/cmd/root.go` (signal context, exit codes, slog level)

## 3. CLI and configuration (`cli-sync`)

- [ ] 3.1 Wire cobra/pflag with imapsync-style flags: host/user/password/port for both sides (passwords as `secret.String`)
- [ ] 3.2 Implement SSL/TLS flags (`--ssl1/--nossl1`, `--tls1/--notls1`, same for side 2) with secure defaults
- [ ] 3.3 Implement `--dry`, `--justfolders`, `--skipemptyfolders`, `--useheader`, `--logfile`, `--version`, `--help`
- [ ] 3.4 Validate required options; support password env fallbacks; never log secrets
- [ ] 3.5 Unit tests for config parsing and validation error cases

## 4. IMAP client layer (`imap-client`)

- [ ] 4.1 Add emersion go-imap (v2) + go-message deps (same major line as go-getmail) behind `internal/imapclient`
- [ ] 4.2 Implement connect, TLS/SSL, login, logout (from go-getmail imap patterns); two independent sessions (host1, host2)
- [ ] 4.3 Implement LIST, CREATE, SELECT/EXAMINE, hierarchy separator detection (new vs go-getmail)
- [ ] 4.4 Implement UID FETCH (headers/body/flags/internaldate) and APPEND with flags + date (APPEND is new)
- [ ] 4.5 Implement reconnect with backoff and re-SELECT; keep secret redaction in all logs
- [ ] 4.6 Unit tests with interface mocks for connect/auth/folder/message ops

## 5. Message identity (`message-sync` foundation)

- [ ] 5.1 Implement `internal/identity` key from Message-Id + Received (configurable `--useheader`) — **not** go-getmail UID keys
- [ ] 5.2 Normalize headers for stable keys; cover missing Message-Id behavior in tests
- [ ] 5.3 Fixtures under `testdata/` for duplicate and unique messages (no secrets)

## 6. Folder and message sync orchestration

- [ ] 6.1 Implement folder discovery and host1→host2 name mapping (separator mapping)
- [ ] 6.2 Implement folder create path honoring `--dry`, `--justfolders`, `--skipemptyfolders`; never delete INBOX
- [ ] 6.3 Build host2 identity set per folder; skip duplicates; FETCH+APPEND new messages
- [ ] 6.4 Preserve system flags and internal date on APPEND
- [ ] 6.5 Track per-folder transferred/skipped/failed counters for the report package
- [ ] 6.6 Integration-style tests with mocked IMAP for full loop and second-run no-op

## 7. Reporting and exit codes (`sync-report`)

- [ ] 7.1 Implement slog setup (stderr + optional `--logfile`) with secret redaction (go-getmail style)
- [ ] 7.2 End-of-run summary: folders, transferred, skipped, errors, success phrase
- [ ] 7.3 Map hard/soft failures to exit codes (auth fail non-zero; clean zero; align with go-getmail 0/1/2 where sensible)
- [ ] 7.4 Tests for summary formatting and exit code policy

## 8. Test harness Docker and Vagrant (`test-harness`)

- [ ] 8.1 Add multi-stage `deploy/docker/Dockerfile` building static `go-imapsync` (no secrets)
- [ ] 8.2 Document `docker build` / run examples with env-passed credentials only
- [ ] 8.3 Add `deploy/vagrant/Vagrantfile` for Ubuntu 24.04: provision Go (or use host binary), run `go test ./...`
- [ ] 8.4 Add `.env.example` with placeholders for live IMAP (host, users, passwords)
- [ ] 8.5 Document live dry-run against `mail.linuxpro.com.br` (opt-in env gate; not default tests)
- [ ] 8.6 Verify default `go test ./...` never contacts external IMAP hosts

## 9. Live validation (operator credentials later)

- [ ] 9.1 Script or Make target `live-dry` that reads env and runs `--dry --justfolders` then `--dry` full
- [ ] 9.2 When credentials for `mail.linuxpro.com.br` are provided: run dry-run, then controlled real sync between agreed mailboxes
- [ ] 9.3 Compare behavior notes with Perl `base/imapsync` on the same accounts (folder list, duplicate skip)
- [ ] 9.4 Record results and known server quirks in a short `docs/live-test.md` (no secrets)

## 10. Polish and handoff

- [ ] 10.1 README: install, flag table for MVP, Docker/Vagrant, live test, attribution to imapsync + go-getmail
- [ ] 10.2 Ensure `go test ./...` and `make build` succeed on host and documented harness
- [ ] 10.3 Mark tasks complete; prepare for `/opsx:apply` follow-ups or archive when acceptance criteria met
