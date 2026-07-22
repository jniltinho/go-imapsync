## ADDED Requirements

### Requirement: Core credentials flags
The CLI SHALL accept source and destination IMAP credentials as long flags compatible with imapsync naming: `--host1`, `--user1`, `--password1`, `--host2`, `--user2`, `--password2`. Port overrides `--port1` and `--port2` MUST be supported. Missing required host/user for a side SHALL cause a non-zero exit before any network sync work.

#### Scenario: Minimal successful invocation shape
- **WHEN** the operator runs the binary with host1/user1/password1 and host2/user2/password2 (and servers accept login)
- **THEN** the tool attempts a one-way sync from host1 to host2 and exits with a defined status code

#### Scenario: Missing host1
- **WHEN** the operator omits `--host1`
- **THEN** the tool prints a usage or validation error and exits non-zero without syncing messages

### Requirement: TLS and SSL connection options
The CLI SHALL provide `--ssl1`/`--nossl1`, `--ssl2`/`--nossl2`, and `--tls1`/`--notls1`, `--tls2`/`--notls2` (or equivalent documented defaults) so operators can force or disable encrypted connections. Default behavior MUST prefer secure connections when the server supports them, without silently disabling certificate verification.

#### Scenario: Explicit SSL on both sides
- **WHEN** the operator passes flags requesting SSL on host1 and host2
- **THEN** the client connects using SSL/TLS to the configured ports (default IMAPS 993 unless overridden)

### Requirement: Dry-run and just-folders modes
The CLI SHALL support `--dry` (no permanent changes on host2: no APPEND of messages and no folder create unless design documents a dry simulation only) and `--justfolders` (folder operations only, no message body transfer). Combining `--dry` and `--justfolders` MUST only report what would be done.

#### Scenario: Dry-run does not append
- **WHEN** the operator runs with `--dry` against real servers
- **THEN** the tool reports planned actions and MUST NOT APPEND messages to host2

#### Scenario: Just folders
- **WHEN** the operator runs with `--justfolders` without `--dry`
- **THEN** the tool may create missing folders on host2 and MUST NOT transfer message bodies

### Requirement: Help and version
The CLI SHALL expose `--help` (or `-h`) listing primary flags and `--version` reporting the build/version string.

#### Scenario: Version flag
- **WHEN** the operator runs with `--version`
- **THEN** the tool prints a version identifier and exits zero without connecting to IMAP

### Requirement: Password via environment
The CLI SHOULD accept passwords from environment variables when flags are omitted (documented names), and MUST NEVER print password values in logs or summaries.

#### Scenario: Password not leaked
- **WHEN** a sync runs with passwords provided
- **THEN** log and summary output do not contain the password strings
