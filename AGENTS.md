# AGENTS.md

## Project Overview

`pgbackrest_exporter` is a Prometheus exporter that collects metrics from
`pgbackrest info --output json` and exposes them over HTTP.
The main entrypoint is `pgbackrest_exporter.go`, which defines the Kingpin CLI,
configures logging and the exporter-toolkit web server, and schedules collection.

The exporter can run directly on a host with pgBackRest, as a systemd service,
or in the Docker image built by this repository.

## Project Structure

- `pgbackrest_exporter.go` contains CLI flags, application startup, signal handling,
  and the collection loop.
- `backrest/` contains pgBackRest command execution and JSON parsing,
  collector configuration, metric definitions, converters, and unit tests.
- `docker_files/run_exporter.sh` maps Docker environment variables to CLI flags.
- `e2e_tests/` contains the Docker-based PostgreSQL and pgBackRest test environment,
  assertions, web configuration fixtures, and test-only certificates and keys.
- `Dockerfile`, `Dockerfile.artifacts`, and `Makefile` are the main build surfaces.
- `.github/workflows/build.yml` runs unit tests, lint, e2e tests, image publishing,
  and GoReleaser.
- `vendor/` is committed dependency source; do not edit vendored files by hand.

## Build, Test, and Lint

Prefer existing Makefile targets over equivalent raw commands.
The project and CI use Go 1.26 and vendored modules.

- Run unit tests with `make test`.
  The target runs:

  ```bash
  TZ="Etc/UTC" go test -mod=vendor -timeout=60s -count 1 ./...
  ```

- For focused iteration, preserve the same flags and select a package and test:

  ```bash
  TZ="Etc/UTC" go test -mod=vendor -timeout=60s -count 1 ./backrest -run TestName
  ```

- `make build` runs unit tests and builds a Linux AMD64 binary.
- `make build-arm` runs unit tests and builds a Linux ARM64 binary.
- `make build-darwin` runs unit tests and builds a Darwin ARM64 binary.
- `make docker` and `make docker-alpine` build the default and Alpine image variants.
- `make dist` builds snapshot archives and packages through Docker and GoReleaser,
  then writes them under the ignored `dist/` directory.
- Run lint: `make lint`.
- Format Go files: `make fmt`.

Every Make invocation evaluates `DOCKER_CONTAINER_E2E` with `docker ps` at parse time,
including `make -n` and targets that otherwise only run Go commands.
A Docker socket or daemon warning from this lookup does not mean e2e tests ran.

## End-to-End Tests

Run the complete suite with:

```bash
make test-e2e
```

The target requires Docker and network access, pulls and builds an e2e image,
creates PostgreSQL and pgBackRest data inside a container, runs all HTTP/TLS/auth cases,
and repeatedly force-removes the `pgbackrest_exporter_e2e` container.
It may first remove a pre-existing container matched by that name.

Do not run `make test-e2e` unless the user explicitly requests it or approves it.
The certificates and private keys under `e2e_tests/` are committed test fixtures only.
Never reuse them for real services or replace them with real credentials.

Keep `e2e_tests/README.md`, the e2e scripts, fixtures, and Makefile e2e cases
synchronized when e2e behavior or coverage changes.

## Verification

- For Go changes, run `make fmt`, `make test`, and `make lint`.
- Focused tests are useful during iteration, but do not use them as the only final verification.
- For build, Docker, or packaging changes, run the relevant Makefile target when practical.
- Run `make test-e2e` only under the approval rules described above.
- For documentation-only changes, do not run code tests unless needed to validate a documented command.
- Always run `git diff --check` before handing off changes.

## Code Style and Test Patterns

- Keep CLI definition and process lifecycle in `pgbackrest_exporter.go`.
- Keep pgBackRest parsing, collection behavior, and metric logic in `backrest/`.
- Prefer simple, concrete changes within the existing package boundaries.
  Add an abstraction, interface, or wrapper only when it removes real duplication,
  reduces existing complexity, or provides a focused test seam.
- Follow the existing standard-library test style: `testing`, table-driven cases,
  `t.Run`, and direct fatal or error assertions.
- Metric tests use an isolated `prometheus.NewRegistry()` and reset package-level metrics
  where needed to prevent state from leaking between cases.
- For code that invokes pgBackRest, use the existing `execCommand` seam and helper-process
  fake pattern, and restore the seam after each test.
- Add focused regression coverage for changed parsing, labels, metric values,
  command arguments, and error behavior.
- Do not introduce `testify` or another test framework solely for convenience.
- For dependency changes, update `go.mod` and `go.sum`, then regenerate committed
  dependency sources with `go mod vendor`.

## Keep Synchronized

- When CLI flags change, update their Kingpin definitions, relevant unit tests,
  README flag documentation and examples, and, when the flag is exposed in Docker,
  `Dockerfile`, `docker_files/run_exporter.sh`, and relevant e2e coverage.
- When metrics or compatibility semantics change, update the definitions and tests
  under `backrest/`, the README metric tables and compatibility notes,
  and e2e metric assertions when they cover the changed behavior.
- Keep `BACKREST_VERSION` and `DOCKER_BACKREST_VERSION` aligned across `Makefile`,
  `Dockerfile`, `e2e_tests/Dockerfile`, and `.github/workflows/build.yml`.
  Keep the user-visible pgBackRest default in `README.md` aligned as well.
- Keep the Go version aligned across `go.mod`, `Dockerfile`, `e2e_tests/Dockerfile`,
  and `.github/workflows/build.yml`.
- Keep the GoReleaser version aligned between `Dockerfile.artifacts`
  and `.github/workflows/build.yml`.

## Runtime and Safety Rules

- `make prepare-service` overwrites the ignored local `pgbackrest_exporter.service`
  from the template and substitutes the current repository path.
  Review the generated unit before installing it.
- `make install-service` writes to `/etc/systemd/system`, reloads systemd,
  enables the exporter, and restarts it.
- `make remove-service` stops and disables the exporter, removes its systemd unit,
  and reloads systemd.
- Never run `make install-service` or `make remove-service` without explicit user approval.
- Do not publish images, packages, tags, or releases unless the user explicitly requests it.

## Generated, Local, and Release Artifacts

- Do not commit ignored local artifacts such as the built `pgbackrest_exporter` binary,
  `pgbackrest_exporter.service`, `dist/`, `.vscode/`, `.DS_Store`, or coverage and trace `*.out` files.
- Keep the committed `vendor/` directory consistent with `go.mod` and `go.sum`.
- GoReleaser has changelog generation disabled; do not add changelog work
  unless a release task explicitly requires it.

## Editing These Notes

Keep this file concise, operational, and limited to repository-specific facts.
Use semantic line breaks and keep one sentence per line where practical
so future changes produce small, reviewable diffs.
