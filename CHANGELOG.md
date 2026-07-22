# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

—

## [0.1.3] — 2026-07-22

**Accurate `--dry` remaining counts after a real sync.**

### Fixed

- fix(sync): build host2 identity keys during `--dry` (SELECT + FETCH headers) so skipped vs pending counts match reality after a completed transfer

## [0.1.2] — 2026-07-22

**Multi-platform release packages and operator good practices.**

### Added

- Cross-build `make release-cross`: **linux/amd64**, **darwin/arm64** (Apple Silicon), **windows/amd64**
- GitHub Actions release uploads all three archives
- docs: `docs/GOOD_PRACTICES.md` (operator staged runs, quota, secrets; developer Go conventions)

## [0.1.1] — 2026-07-22

**Clearer operator errors and fuller package godoc.**

### Added

- `internal/imaperr`: classify quota, closed connection, auth, TLS, timeout, and related IMAP failures with hints
- End-of-run summary error breakdown and “what to do next” hints
- CONTRIBUTING.md and expanded godoc on packages and exported APIs

### Changed

- Abort folder/run on OVERQUOTA or repeated closed-connection APPEND failures instead of spamming identical errors
- Release notes style aligned with go-postfixadmin (emoji sections + Full Changelog)

## [0.1.0] — 2026-07-22

**First public MVP:** one-way IMAP mailbox sync (host1 → host2) as a static Go binary.

### Added

- CLI with imapsync-style flags (`--host1/2`, SSL/TLS, `--dry`, `--justfolders`, `--useheader`, `--logfile`, …)
- Dual-host IMAP client on `emersion/go-imap/v2` (LIST, CREATE, FETCH, APPEND with flags and internal date)
- Header-based duplicate detection (default Message-Id + Received)
- Password redaction (`internal/secret`) and optional env passwords
- End-of-run summary and exit codes 0/1/2
- Unit tests, dual-user TLS memserver integration tests, and binary e2e smoke test
- Makefile (`build`, `test`, `lint`, `release`, `docker-build`, `live-dry`)
- GitHub Actions CI and tag-based release workflow
- Docker multi-stage image
- OpenSpec change `go-imapsync-core` (proposal, design, specs, tasks)
- Archify architecture diagrams under `docs/architecture/`

### Notes

- Full imapsync option parity (XOAUTH2, `--delete*`, folder mapping, etc.) is intentionally out of scope for 0.1.0
- Live operator tests use credentials at runtime only; `docs/prints/` and `base/` are not published
