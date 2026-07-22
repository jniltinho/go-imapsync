## Purpose

Incremental message transfer with header-based identity and duplicate skip.

## Requirements

### Requirement: Header-based message identity
The system SHALL identify whether a host1 message already exists on host2 by comparing an identity key derived from configurable header fields. The default headers MUST be Message-Id and Received (imapsync-compatible). The tool SHALL support selecting headers via `--useheader` (at least Message-Id alone).

#### Scenario: Skip duplicate by Message-Id
- **WHEN** a message with the same identity key already exists in the corresponding host2 folder
- **THEN** the tool does not APPEND a duplicate copy of that message

#### Scenario: Transfer new message
- **WHEN** a host1 message identity is not present on host2 for that folder sync context
- **THEN** the tool FETCHes and APPENDs the message (unless `--dry`)

### Requirement: Incremental restartable sync
The system SHALL be safe to re-run: a second run without new host1 mail MUST transfer zero new messages (aside from any previously failed items still missing).

#### Scenario: Second run is no-op for messages
- **WHEN** a successful full sync is immediately repeated with the same options
- **THEN** transferred-new count is zero and existing messages remain on host2

### Requirement: Preserve flags and internal date
The system SHALL copy standard IMAP system flags to host2 and SHOULD set the host2 internal date from host1 INTERNALDATE when the APPEND API allows it.

#### Scenario: Unread stays unread
- **WHEN** a message without `\Seen` is transferred
- **THEN** the host2 copy is not marked `\Seen` solely because of the transfer

### Requirement: Per-folder progress
The system SHALL process messages per folder and record counts of transferred, skipped (already present), and failed messages for the report capability.

#### Scenario: Counts after mixed folder
- **WHEN** a folder has both new and already-synced messages
- **THEN** the run records non-zero skipped and transferred counts accordingly

### Requirement: No size-as-identity
The system MUST NOT use message size alone as the identity key for duplicate detection.

#### Scenario: Same content different reported size
- **WHEN** two servers report different RFC822.SIZE for the same Message-Id identity
- **THEN** the message is still treated as already present if the header identity matches
