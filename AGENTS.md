# Repository Guidelines

## Project Structure & Module Organization
- `sqliterouter.go` defines the Caddy HTTP handler that loads SQLite routes and exposes `backend_upstream` for reverse proxies.
- `sqliterouter_test.go` covers handler behavior with table-driven tests; place any additional Go tests alongside the code they verify.
- `mkdb.py` and `test_e2e.py` live in the repo root with helper assets such as `Caddyfile_test`; keep Python utilities here so CI can invoke them directly.
- Dependencies are tracked in `go.mod`/`go.sum`, while `README.md` documents top-level usage and should be updated when behavior changes.

## Build, Test, and Development Commands
Run `xcaddy build --with github.com/AnswerDotAI/caddy-sqlite-router` to produce a Caddy binary that bundles this module.
After installing Go 1.25+, use `go test -v ./...` for unit coverage and `go vet ./...` before pushing.
For end-to-end checks:
```bash
python -m venv .venv && source .venv/bin/activate
pip install -r test_requirements.txt
python mkdb.py && python test_e2e.py
```
This seeds the sample database and exercises the router against a local Caddy instance.

## Testing Guidelines
Add unit tests under `*_test.go`, mirroring the package under test. Update `mkdb.py` fixtures to cover new database columns, and extend `test_e2e.py` when the public behavior changes. Run unit tests plus the end-to-end script before opening a PR.

## Commit & Pull Request Guidelines
Favor short, imperative subjects (e.g., `add sql named function`).

