## 1. Project scaffolding (aligned with go-getmail)

- [x] 1.1 Initialize Go module at repo root (`go.mod`); module path `go-imapsync`, Go 1.26+
- [x] 1.2 Create package layout: `cmd/`, `internal/{config,secret,imapclient,identity,sync,report}` (+ `imaperr`; no separate `logx` — slog wired in `cmd`)
- [x] 1.3 Add README (build, flags, refs to imapsync + go-getmail, no secrets)
- [x] 1.4 Add MIT LICENSE and `.gitignore` (binaries, `.env`, logs, `base/`, `docs/prints/`)
- [x] 1.5 Add `Makefile`: `build`, `test`, `lint`, `docker-build`, `release`, `release-cross`, `live-dry`

## 2. Port reusable pieces from go-getmail

- [x] 2.1 Adapt `internal/secret` (redacted String, LogValue, FromCommand, tests)
- [x] 2.2 Dual-host client from go-getmail IMAP patterns → `internal/imapclient`
- [x] 2.3 Cobra root Execute (signal context, exit codes 0/1/2, slog)

## 3. CLI and configuration (`cli-sync`)

- [x] 3.1 imapsync-style flags host/user/password/port both sides (`secret.String`)
- [x] 3.2 SSL/TLS flags with secure defaults (IMAPS 993)
- [x] 3.3 `--dry`, `--justfolders`, `--skipemptyfolders`, `--useheader`, `--logfile`, `--version`, `--help`
- [x] 3.4 Validate required options; env password fallbacks; never log secrets
- [x] 3.5 Unit tests for config validation

## 4. IMAP client layer (`imap-client`)

- [x] 4.1 emersion go-imap/v2 behind `internal/imapclient`
- [x] 4.2 Connect, TLS/SSL, login, logout; two independent sessions
- [x] 4.3 LIST, CREATE, SELECT/EXAMINE, hierarchy delimiter
- [x] 4.4 UID FETCH (headers/body/flags/internaldate) and APPEND with flags + date
- [x] 4.5 Connection loss: classified errors + abort (full reconnect/backoff deferred to later change)
- [x] 4.6 Unit/integration tests via dual-user TLS memserver (`internal/testutil`)

## 5. Message identity (`message-sync` foundation)

- [x] 5.1 `internal/identity` Message-Id + Received (`--useheader`)
- [x] 5.2 Normalize headers; missing Message-Id covered in tests
- [x] 5.3 Identity tests with inline fixtures (no secrets; dedicated `testdata/` optional later)

## 6. Folder and message sync orchestration

- [x] 6.1 Folder discovery + host1→host2 separator mapping
- [x] 6.2 Folder create honors `--dry` / `--justfolders`; never delete INBOX (`--skipemptyfolders` create-skip still partial)
- [x] 6.3 Host2 identity set; skip duplicates; FETCH+APPEND; dry-run also compares host2
- [x] 6.4 Preserve system flags and internal date on APPEND
- [x] 6.5 Transferred/skipped/failed counters + error classification for report
- [x] 6.6 Integration tests memserver: transfer, second-run skip, dry, justfolders

## 7. Reporting and exit codes (`sync-report`)

- [x] 7.1 slog stderr + optional `--logfile`; secret redaction
- [x] 7.2 End-of-run summary: folders, transferred, skipped, errors, success phrase; breakdown + hints
- [x] 7.3 Exit codes 0/1/2 (auth/runtime vs usage)
- [x] 7.4 Tests for summary and error recording

## 8. Test harness Docker and Vagrant (`test-harness`)

- [x] 8.1 Multi-stage `deploy/docker/Dockerfile` (no secrets)
- [x] 8.2 Document docker build/run (README)
- [ ] 8.3 Vagrant Ubuntu 24.04 harness — **deferred** (Docker + memserver e2e cover automated path)
- [x] 8.4 `.env.example` with placeholders (`mail.orig-domain.com` / `mail.dest-domain.com`)
- [x] 8.5 Live dry-run docs (README + GOOD_PRACTICES + `make live-dry`)
- [x] 8.6 Default `go test ./...` never contacts external IMAP

## 9. Live validation (operator credentials later)

- [x] 9.1 `make live-dry` (env credentials)
- [x] 9.2 Live dry + real sync validated (criare-net → linuxpro; resume after quota; final dry: 0 pending / 14467 skipped)
- [ ] 9.3 Formal side-by-side notes vs Perl `base/imapsync` — **deferred**
- [ ] 9.4 `docs/live-test.md` — **deferred** (results captured in session; GOOD_PRACTICES covers operator checklist)

## 10. Polish and handoff

- [x] 10.1 README: install, flags, multi-platform releases, Docker, live, architecture, attribution
- [x] 10.2 `go test ./...` and `make build` / `release-cross` green; releases v0.1.0–v0.1.3
- [x] 10.3 Mark tasks complete; archive OpenSpec change when accepted

### Deferred (follow-up changes)

| Item | Reason |
|------|--------|
| IMAP reconnect/backoff | Abort + classify only in MVP |
| Vagrant Ubuntu 24.04 | Optional; Docker + e2e sufficient for now |
| Full `--skipemptyfolders` CREATE skip | Partial |
| Formal Perl imapsync comparison doc | Optional ops note |
| `docs/live-test.md` | Covered by GOOD_PRACTICES + session logs |
