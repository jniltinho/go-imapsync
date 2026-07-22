## ADDED Requirements

### Requirement: Unit and package tests
The repository SHALL include automated Go tests for identity key generation, config validation, and sync orchestration with mocked IMAP clients. `go test ./...` MUST pass without network access and without live credentials.

#### Scenario: Offline test suite
- **WHEN** a developer runs `go test ./...` with no IMAP credentials configured
- **THEN** all default tests pass without contacting external hosts

### Requirement: Docker build path
The project SHALL provide a Dockerfile that builds the `go-imapsync` binary. Building the image MUST not require embedding live secrets.

#### Scenario: Docker image builds
- **WHEN** an operator runs the documented Docker build
- **THEN** an image is produced containing a runnable go-imapsync binary

### Requirement: Vagrant Ubuntu 24.04 path
The project SHALL provide a Vagrant-based environment targeting Ubuntu 24.04 (or documented equivalent) that can install dependencies, build the project, and run the offline test suite.

#### Scenario: Vagrant provision and test
- **WHEN** an operator runs `vagrant up` (with provider available) and the documented test command inside the VM
- **THEN** the offline tests pass inside the guest

### Requirement: Live IMAP test against mail.linuxpro.com.br
The project SHALL document how to run a live dry-run and optional real sync against `mail.linuxpro.com.br` using credentials supplied only at runtime (environment variables or flags). Live tests MUST be opt-in and MUST NOT run in the default `go test ./...` suite.

#### Scenario: Live dry-run gated
- **WHEN** live credentials are not set
- **THEN** default CI/unit tests do not attempt to connect to mail.linuxpro.com.br

#### Scenario: Live dry-run with credentials
- **WHEN** the operator sets the documented credential environment variables and runs the documented live dry-run command against mail.linuxpro.com.br
- **THEN** the tool connects, authenticates, and completes a `--dry` pass without appending messages

### Requirement: Secrets hygiene in harness
Dockerfiles, Vagrantfiles, scripts, and testdata MUST NOT contain real passwords or tokens. Examples MUST use placeholders only.

#### Scenario: No secrets in repo harness files
- **WHEN** harness files under deploy/ and example env files are inspected
- **THEN** they contain only placeholders or empty values for secrets
