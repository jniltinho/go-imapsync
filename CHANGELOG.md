# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

—

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
