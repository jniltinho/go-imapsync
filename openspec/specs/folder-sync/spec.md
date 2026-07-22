## Purpose

Folder hierarchy discovery and creation on the destination.

## Requirements

### Requirement: Recursive folder hierarchy
The system SHALL discover the folder hierarchy on host1 and process folders for sync to host2, preserving hierarchy using the destination hierarchy separator (mapping separators when host1 and host2 differ).

#### Scenario: Nested folder copied
- **WHEN** host1 has a nested folder (e.g. parent/child) and message or folder sync runs without filters excluding it
- **THEN** the corresponding hierarchy exists on host2 after a successful non-dry folder sync

### Requirement: Skip empty folders option
The system SHALL support skipping creation of empty host1 folders on host2 when `--skipemptyfolders` is set. Without that flag, empty folders MAY still be created to mirror structure (document default; recommend creating structure unless skipped).

#### Scenario: Skip empty
- **WHEN** `--skipemptyfolders` is set and a host1 folder contains no messages
- **THEN** the tool does not create that folder on host2 solely for structure

### Requirement: Folder name safety for INBOX
The system MUST NOT delete or rename INBOX on host2 as part of folder synchronization in this capability scope.

#### Scenario: INBOX preserved
- **WHEN** folder sync runs
- **THEN** host2 INBOX remains present and is not removed by go-imapsync

### Requirement: Just-folders orchestration
When `--justfolders` is active, the folder-sync path SHALL run folder discovery and optional creates, and MUST NOT enter message body transfer.

#### Scenario: Justfolders skips messages
- **WHEN** `--justfolders` is set
- **THEN** no message APPEND operations are performed
