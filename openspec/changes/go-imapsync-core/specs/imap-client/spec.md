## ADDED Requirements

### Requirement: Dual independent IMAP sessions
The system SHALL maintain two independent IMAP sessions (host1 source and host2 destination). Implementation MAY adapt dial/login/select/fetch patterns from the go-getmail reference (`base/go-getmail/internal/retriever/imap.go`) but MUST support concurrent logical roles without sharing one authenticated client for both hosts.

#### Scenario: Two logins
- **WHEN** a sync starts with valid credentials for both hosts
- **THEN** the tool establishes separate authenticated sessions for host1 and host2

### Requirement: Connect and authenticate
The system SHALL open IMAP sessions to host1 and host2 using the configured host, port, TLS/SSL mode, username, and password. Authentication failure MUST be reported clearly and abort the sync with non-zero exit.

#### Scenario: Successful login both sides
- **WHEN** valid credentials are provided for host1 and host2
- **THEN** both sessions are authenticated and ready for folder commands

#### Scenario: Invalid password on host1
- **WHEN** host1 rejects authentication
- **THEN** the tool fails fast with an error mentioning host1 authentication and does not claim a successful sync

### Requirement: Folder list select and create
The IMAP client layer SHALL list mailboxes on host1, SELECT/EXAMINE folders as needed, and CREATE missing folders on host2 when sync requires them (unless dry-run forbids creates). INBOX MUST be treated as a mandatory folder that is never deleted by this tool in the MVP.

#### Scenario: Create missing destination folder
- **WHEN** host1 has a folder that does not exist on host2 and message or folder sync requires it
- **THEN** the client creates the folder on host2 (when not in dry-run)

### Requirement: Fetch and append messages
The client SHALL FETCH message content and flags from host1 (UID-based when available) and APPEND to host2 with flags and internal date when supported by the server.

#### Scenario: Append preserves Seen flag
- **WHEN** a message with `\Seen` is transferred
- **THEN** the message on host2 is stored with `\Seen` if the server accepts that flag

### Requirement: Reconnect on transient failure
The client SHALL attempt reconnection with backoff after transient network or IMAP errors, re-establish folder selection, and continue from a safe point (next message or retry current once). Permanent errors after retry budget MUST surface to the sync report.

#### Scenario: Transient disconnect mid-folder
- **WHEN** the connection to host2 drops mid-folder and recovers within the retry budget
- **THEN** the tool reconnects, reselects the folder, and continues transferring remaining messages without aborting the whole run solely due to that single drop

### Requirement: No password logging
The IMAP client and logger MUST redact or omit credentials from debug output even when IMAP command tracing is enabled.

#### Scenario: Debug IMAP without secrets
- **WHEN** verbose/debug IMAP logging is enabled
- **THEN** password material is not written to the log stream
