# go-imapsync

A Go reimplementation of the essentials of [imapsync](https://imapsync.lamiral.info/):
**one-way**, incremental IMAP mailbox sync from **host1 → host2**, without duplicates,
as a single static binary with no runtime dependencies.

> **Attribution**
>
> - **Behavior reference:** Perl imapsync (Gilles LAMIRAL). No Perl code was copied;
>   documentation and flags guided the design.
> - **Go engineering base:** [jniltinho/go-getmail](https://github.com/jniltinho/go-getmail)
>   (Cobra CLI, `emersion/go-imap/v2`, secret redaction, Makefile/CI patterns).

## Install / Build

```bash
make build            # static binary dist/go-imapsync
make release          # build + compress with upx (optional, if installed)
make test             # go test -race ./...
make lint             # gofmt + go vet + staticcheck
```

Requires Go 1.26+.

## Usage

```bash
go-imapsync \
  --host1 imap.source.example --user1 alice --password1 'secret1' \
  --host2 imap.dest.example   --user2 bob   --password2 'secret2'

# Safe first steps (like imapsync examples):
go-imapsync ... --justfolders --dry
go-imapsync ... --justfolders
go-imapsync ... --dry
go-imapsync ...

go-imapsync version
go-imapsync --help
```

**Good practices** (quota, staged runs, secrets, resume): see [docs/GOOD_PRACTICES.md](docs/GOOD_PRACTICES.md).

Passwords may also come from the environment (never logged):

- `GOIMAPSYNC_PASSWORD1`
- `GOIMAPSYNC_PASSWORD2`

Exit codes: `0` success · `1` runtime failure · `2` usage/config error.

### Common flags

| Flag | Meaning |
|------|---------|
| `--host1/--user1/--password1/--port1` | Source account |
| `--host2/--user2/--password2/--port2` | Destination account |
| `--ssl1/--nossl1` · `--ssl2/--nossl2` | IMAPS (default on, port 993) |
| `--tls1/--notls1` · `--tls2/--notls2` | STARTTLS when not using SSL |
| `--dry` | Report actions only; no CREATE/APPEND on host2 |
| `--justfolders` | Folders only, no message bodies |
| `--skipemptyfolders` | Do not mirror empty folders |
| `--useheader` | Identity headers (default `Message-Id`, `Received`) |
| `--logfile` | Also write logs to a file |
| `--verbose` | Debug logging |
| `--insecuretls` | Skip TLS verify (**lab only**) |
| `--timeout` | Network timeout (default 60s) |

Duplicate detection uses **headers** (imapsync model), **not** IMAP UIDs
(go-getmail’s oldmail model is intentionally different).

## Docker

```bash
make docker-build
docker run --rm go-imapsync:dev version
# Pass secrets via env or flags at runtime — never bake them into the image.
```

## Live test (opt-in)

Against real IMAP servers, credentials only at runtime:

```bash
export GOIMAPSYNC_HOST1=mail.orig-domain.com
export GOIMAPSYNC_USER1=...
export GOIMAPSYNC_PASSWORD1=...
export GOIMAPSYNC_HOST2=mail.dest-domain.com
export GOIMAPSYNC_USER2=...
export GOIMAPSYNC_PASSWORD2=...
make live-dry
```

Default `make test` never contacts external IMAP hosts.

## Architecture

Interactive diagrams (HTML, dark/light theme, PNG/SVG export):

| Diagram | Path |
|---------|------|
| System architecture | [docs/architecture/go-imapsync-architecture.html](docs/architecture/go-imapsync-architecture.html) |
| Sync run workflow | [docs/architecture/sync-run-workflow.html](docs/architecture/sync-run-workflow.html) |
| Message transfer sequence | [docs/architecture/message-transfer-sequence.html](docs/architecture/message-transfer-sequence.html) |
| Message data path | [docs/architecture/message-path-dataflow.html](docs/architecture/message-path-dataflow.html) |

See [docs/architecture/README.md](docs/architecture/README.md) for sources and re-render instructions.

## Project layout

```text
cmd/                 # Cobra CLI
internal/config/     # flag-derived options
internal/secret/     # redacted passwords (from go-getmail)
internal/imapclient/ # dual-host IMAP (go-imap/v2)
internal/identity/   # Message-Id + Received keys
internal/sync/       # folder + message orchestration
internal/report/     # end-of-run summary
deploy/docker/       # Dockerfile
docs/architecture/   # Archify HTML diagrams + JSON sources
examples/            # sample invocations
openspec/            # OpenSpec design (go-imapsync-core)
```

## Releases

Tagged releases (`v*`) are built by GitHub Actions (linux/amd64 tarball).  
See [CHANGELOG.md](CHANGELOG.md) and [GitHub Releases](https://github.com/jniltinho/go-imapsync/releases).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for setup, tests, and PR expectations.

## Status (MVP)

Supported:

- SSL/IMAPS and STARTTLS
- Recursive folder LIST + CREATE on host2
- Message FETCH/APPEND with flags and internal date
- Header-based skip of duplicates
- `--dry`, `--justfolders`, summary + exit codes

Later (see `openspec/changes/go-imapsync-core/`): folder filters/mapping, `--delete1/2`,
XOAUTH2, Gmail labels, mass CSV, etc.

## License

[MIT](LICENSE).
