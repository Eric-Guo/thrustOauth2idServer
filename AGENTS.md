# Repository Guidelines

## Project Structure & Module Organization
- The Go module `thrust_oauth2id` boots from `cmd/thrustOauth2idServer/main.go`; keep CLI or server wiring here only.
- Core packages sit in `internal/`: `routers` registers Gin routes, `handler` hosts HTTP logic, `dao` and `model` back storage, while `server` handles lifecycle; add new business logic under a focused subpackage such as `internal/service`.
- Environment assets live alongside the code: configs in `configs/`, deployment manifests in `deployments/`, docs and Swagger sources in `docs/`, helper scripts in `scripts/`; mirror this layout when adding files.
- Tests belong next to their targets (`internal/handler/user_test.go` style) so ownership stays obvious.

## Build, Test, and Development Commands
- `make run [Config=configs/dev.yml]` compiles and starts the server with an optional config override.
- `make build` produces a deployable binary in `cmd/thrustOauth2idServer/`.
- `make ci-lint` runs `gofmt -s` and `golangci-lint`; run it before pushing or opening a PR.
- `make docs` regenerates Swagger artifacts after handler or model updates.
- `make test` executes fast unit suites; `make cover` adds HTML coverage for local review.

## Coding Style & Naming Conventions
- Format all Go files with `gofmt`/`goimports` (module prefix `thrust_oauth2id`) and keep lines under 200 characters per `lll`.
- Follow lints in `.golangci.yml` (`revive`, `staticcheck`, `misspell`, etc.); resolve warnings rather than suppressing them.
- Exported types and functions use UpperCamelCase, package names stay short and lowercase, and structured logs should favor consistent field keys (`request_id`, `subject`, etc.).

## Testing Guidelines
- Author table-driven tests in `*_test.go` files, leaning on `testing` and `net/http/httptest` for handler coverage.
- Stub external resources (SQLite DSNs, Redis) so suites run hermetically; keep fixtures under `testdata/` when needed.
- Run `make test` before every commit and `make cover` for new flows to confirm coverage deltas.

## Commit & Pull Request Guidelines
- Write imperative, concise commit subjects (e.g., `Add Rails Auth`, `Fix token refresh`) with optional bodies for context or follow-up tasks.
- PRs should link related issues, list touched configs, and include manual test evidence or `make` commands executed; attach Swagger screenshots when docs change.
- Request reviewers familiar with the touched package and call out secrets, migrations, or deployment sequencing explicitly.

## Security & Configuration Tips
- Never commit environment secrets; instead copy `configs/thrustOauth2idServer.yml` and inject secrets via CI or runtime tooling.
- When enabling TLS, Redis, or new third-party services, update matching assets in `deployments/` and document operational steps in the PR or `docs/`.
