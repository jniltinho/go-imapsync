## ADDED Requirements

### Requirement: End-of-run summary
At the end of a sync attempt, the system SHALL print a summary including at least: number of folders processed, messages transferred, messages skipped as duplicates, error count, and overall success or failure. The summary MUST be written even when some soft errors occurred.

#### Scenario: Clean sync summary
- **WHEN** a sync completes with zero errors
- **THEN** the summary indicates success and reports transferred and skipped counts

#### Scenario: Errors in summary
- **WHEN** one or more message transfers fail but the process completes
- **THEN** the summary includes a non-zero error count and a non-zero exit code if the failure policy requires it

### Requirement: Exit codes
The system SHALL exit with code 0 only when the run meets the success policy (no hard failures and errors within configured tolerance). Authentication failures and unrecoverable connection failures MUST yield non-zero exit.

#### Scenario: Auth failure exit
- **WHEN** authentication to either host fails
- **THEN** the process exits non-zero

### Requirement: Optional logfile
The system SHALL support writing the run log to a file path via `--logfile` (or equivalent). Without it, logs go to stderr (or documented default). `--nolog` or default MAY omit file logging.

#### Scenario: Logfile created
- **WHEN** the operator passes `--logfile /tmp/go-imapsync-test.log`
- **THEN** a log file is written at that path containing run output without passwords

### Requirement: Operator-friendly success signal
The summary SHALL include a clear phrase indicating whether all identified host1 messages appear present on host2 (similar intent to imapsync “sync looks good”), without requiring byte-identical wording.

#### Scenario: All messages present
- **WHEN** every identified host1 message in scope has a matching identity on host2 after the run
- **THEN** the summary states that the sync looks complete/good for identified messages
