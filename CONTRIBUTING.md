# Contributing

Thanks for improving go-imapsync.

## Prerequisites

- Go 1.26+
- `make`
- Optional: `staticcheck`, `upx`, Docker

## Setup

```bash
git clone git@github.com:jniltinho/go-imapsync.git
cd go-imapsync
make test
make build
./dist/go-imapsync version
```

Local reference trees under `base/` (Perl imapsync, go-getmail) are optional and **gitignored**.

## Development

| Command | Purpose |
|---------|---------|
| `make test` | `go test -race ./...` (includes e2e) |
| `make build` | Static binary in `dist/go-imapsync` |
| `make lint` | gofmt + go vet (+ staticcheck if installed) |
| `go test -short ./...` | Skip e2e smoke |

Do not commit secrets, `.env`, or `docs/prints/`.

## Code style

- Follow Go conventions and the project golang skills under `.agents/skills/` (naming, error handling, godoc, testing).
- Exported identifiers need godoc comments starting with the name.
- Prefer clear errors classified via `internal/imaperr` when talking to IMAP.
- Operator and developer conventions are summarized in [docs/GOOD_PRACTICES.md](docs/GOOD_PRACTICES.md).

## Pull requests

1. Branch from `main`
2. Keep changes focused; update `CHANGELOG.md` under `[Unreleased]` when user-visible
3. Ensure `make test` and `make build` pass
4. Open a PR against `main`

## Releases

Maintainers cut releases with the local create-release skill (GitHub + `gh`). Tags `v*` trigger `.github/workflows/release.yml`.

## License

By contributing you agree that your contributions are licensed under the MIT License (see [LICENSE](LICENSE)).
