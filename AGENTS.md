# Repository Guidelines

This guide helps contributors work effectively on the `simplified-cce` Go CLI.

## Project Structure & Module Organization
- Module: `simplified-cce`; CLI entry at `main.go`.
- Core files in repo root: `launcher.go`, `config.go`, `ui.go`.
- Tests colocated as `*_test.go` (e.g., `main_test.go`).
- CI workflows and templates in `.github/`; lint config in `.golangci.yml`.
- Runtime config: `~/.claude-code-env/config.json`; timestamped backups in `~/.claude-code-env/backups/`.

## Build, Test, and Development Commands
- `make build`: Compile the CLI to `./cce`.
- `make test`: Run all unit tests verbosely.
- `make test-coverage`: Generate `coverage.out` and `coverage.html`.
- `make bench`: Execute benchmarks.
- `make fmt` / `make vet`: Format imports/code and run vet checks.
- `golangci-lint run`: Full lint suite per `.golangci.yml`.
- Example subset: `go test -run TestName ./...`.

## Coding Style & Naming Conventions
- Formatting: `gofmt`/`goimports` (run `make fmt` before committing).
- Imports: group stdlib, third‑party, then local (`simplified-cce/...`).
- Naming: Exported `CamelCase`; unexported `camelCase`.
- Keep functions small; prefer table‑driven tests; avoid naked returns in long funcs.

## Testing Guidelines
- Framework: Go `testing` package; place tests next to sources as `*_test.go`.
- Names: `TestXxx` for tests; `BenchmarkXxx` for benchmarks.
- Coverage: maintain or improve; verify with `make test-coverage`.
- Security: run `make test-security` for security‑focused tests when applicable.

## Commit & Pull Request Guidelines
- Commits: concise prefixes (`feat:`, `fix:`, `chore:`, `docs:`, `refactor:`, `test:`, `style:`); explain the "why" when non‑obvious.
- PRs: clear description, linked issues (e.g., `Closes #123`), tests for changes, and CLI output/screenshots if behavior changes.
- Quality gate: ensure `make quality` (and linters) pass before review.

## Security & Configuration Tips
- Do not commit secrets. Config lives under `~/.claude-code-env/` with `0700/0600` permissions; backups are created before edits (see `config.go`).
- For code touching I/O or crypto, run `gosec` via `golangci-lint` locally.

