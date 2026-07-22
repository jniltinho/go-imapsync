# Good practices

Operator guidance for safe mailbox sync, plus developer conventions used in this
repository. Behavior principles follow classic [imapsync](https://imapsync.lamiral.info/)
(safe defaults, restartable, no harm by default). Engineering practices follow
the Go skills under `.agents/skills/` (code style, naming, errors, testing, docs).

---

## Operator practices

### 1. Prefer a staged run

Always learn the mapping before writing data:

```bash
# 1) Folders only, no changes
go-imapsync ... --justfolders --dry

# 2) Create folders on host2
go-imapsync ... --justfolders

# 3) Full plan, no APPEND
go-imapsync ... --dry

# 4) Real transfer
go-imapsync ...
```

Stop between steps if folder names look wrong.

### 2. Passwords and secrets

| Do | Don’t |
|----|--------|
| Prefer `GOIMAPSYNC_PASSWORD1` / `GOIMAPSYNC_PASSWORD2` | Commit passwords, `.env`, or logs with secrets |
| Keep credentials out of shell history when possible | Use `--insecuretls` on production |
| Rotate passwords if they appeared in chat/logs | Bake secrets into Docker images |

Passwords are redacted in structured logs (`secret.String`). Never log `Reveal()`.

### 3. TLS and ports

- Default is **IMAPS** (SSL on, port **993**).
- Use STARTTLS only when the server requires plain **143** then upgrade (`--nossl1 --tls1`, same for host2).
- Certificate errors usually mean wrong hostname, wrong port, or corporate MITM — fix the trust chain; do not habitually disable verify.

### 4. Destination capacity

Before a large sync:

- Check **quota** on host2 (we have seen live `[OVERQUOTA]` after ~thousands of messages).
- Leave headroom for Sent + INBOX growth.
- If the run aborts with quota or “connection closed”, **free space or raise quota**, then **re-run**. Already copied messages are skipped by header identity.

### 5. Identity and duplicates

- Default identity: **Message-Id** + **Received** (imapsync model), not IMAP UID.
- Re-runs are intentional and safe for incremental catch-up.
- Messages **without** Message-Id may transfer again on later runs until a future `--addheader`-style option exists — prefer fixing source clients or filtering those folders carefully.
- If you see systematic duplicates, try `--useheader Message-Id` alone and compare.

### 6. Interruptions and resume

- Abort with Ctrl+C; wait for a clean summary when possible.
- After network drops, quota, or manual stop: **run the same command again**.
- Stateless design: no local DB is required for resume.

### 7. Logging and support

```bash
go-imapsync ... --verbose --logfile /tmp/go-imapsync-run.log
```

Read the final summary for:

- transferred / skipped / failed counts  
- **Error breakdown** (quota, connection_closed, …)  
- **What to do next** hints  

Share logs only after redacting hosts/users if needed; passwords should never appear.

### 8. What this tool is not

- **Not two-way sync** (use mbsync/offlineimap-style tools for that).
- **Not a delete/migration cleanup tool** in the MVP (`--delete1` / `--delete2` are out of scope for now).
- **Not** a replacement for backup policy — treat host2 as a copy destination you control.

### 9. Live / production checklist

- [ ] Credentials via env or short-lived flags  
- [ ] Dry + justfolders reviewed  
- [ ] Quota on host2 verified  
- [ ] IMAPS endpoints confirmed  
- [ ] Logfile path writable  
- [ ] Plan time for large INBOXes (hours possible)  
- [ ] Re-run strategy understood  

---

## Developer practices

These match the project’s golang skills (style, naming, errors, testing, documentation).

### Code

- **Clear is better than clever** — early returns, small functions, limited nesting.
- **context.Context first** on I/O and sync entry points.
- **Wrap errors with `%w`** and side/op context; classify IMAP faults via `internal/imaperr` before logging.
- **Never log secrets** — use `secret.String`; call `Reveal()` only at LOGIN.
- **Unexport by default**; export only stable API surface.
- **Packages under `internal/`** stay application-private.

### Tests

- Default `make test` / `go test ./...` must stay **offline** (no real IMAP).
- Prefer table-driven tests; use `internal/testutil` dual-user TLS memserver for IMAP paths.
- E2E builds the real binary; skip with `-short` when needed.
- Live operator tests are **opt-in** (`make live-dry` + env credentials).

### Docs

- Every **package** has a package comment.
- Every **exported** name has a godoc starting with that name; explain *why* / *when* / *what fails*, not the signature alone.
- User-visible changes go in `CHANGELOG.md` (`[Unreleased]` first).
- Architecture diagrams live under `docs/architecture/` (English).

### Git and releases

- Do not commit `base/`, `docs/prints/`, `.env`, or binaries under `dist/`.
- Tag releases `v*`; GitHub Actions builds the tarball; enrich notes with `gh` in the go-postfixadmin style (emoji sections + Full Changelog). See the local create-release skill.

### Suggested review order for a PR

1. `make test` and `make build` green  
2. No secrets in the diff  
3. Errors classified where user-visible  
4. Godoc on new exported symbols  
5. CHANGELOG / docs if behavior changed  

---

## Related docs

| Doc | Audience |
|-----|----------|
| [README.md](../README.md) | Install and flags |
| [CONTRIBUTING.md](../CONTRIBUTING.md) | Setup and PRs |
| [CHANGELOG.md](../CHANGELOG.md) | Version history |
| [architecture/](architecture/) | Diagrams |
| OpenSpec `go-imapsync-core` | Design decisions |
