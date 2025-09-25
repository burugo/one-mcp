# Repository Guidelines

## Project Structure & Module Organization
The Go backend lives under `backend/` with request handlers in `api/handler`, shared utilities in `common/`, data access in `data/`, and business logic in `service/`. The React frontend sits in `frontend/src` with translations in `frontend/public/locales` and build artifacts generated into `frontend/dist`. Persistent assets such as the SQLite database and uploads are stored in `data/` and `upload/`, while deployment aides live in `deploy/`, `Dockerfile`, and `docker-compose.yaml`.

## Build, Test, and Development Commands
- `./run.sh` — launches the backend on `:3000` and the Vite dev server on `:5173` with hot reload.
- `PORT=8080 ./build.sh` — produces a production binary and bundles the frontend.
- `go test ./...` — executes the Go unit tests across the backend.
- `cd frontend && npm run build` — type-checks, lints, and compiles the React app.

## Coding Style & Naming Conventions
Go code must pass `gofmt` (tabs for indentation) and follow idiomatic package naming (`lower_case` for directories, `CamelCase` for exported types). Keep API handlers in `backend/api/handler` named `*_handler.go` and tests as `*_test.go`. TypeScript and JSX files use two-space indentation, TypeScript strict mode, and Tailwind utility ordering as emitted by `shadcn`. Run `npm run lint` before pushing to ensure ESLint (flat config) passes.

## Testing Guidelines
Backend features require table-driven tests under matching `*_test.go` files; ensure new logic is covered by `go test ./...` and update `coverage.out` when reporting coverage. Frontend components should use Vitest with Testing Library—add specs under `frontend/src/**/__tests__/` or alongside components with `.test.tsx` suffix. For UI flows, prefer `npm run test:coverage` to exercise V8 coverage and attach results in PRs touching critical paths.

## Commit & Pull Request Guidelines
Follow the Conventional Commits style visible in `git log` (e.g., `feat(proxy): add SSE support`). Keep messages scoped, written in the imperative, and limited to 72 characters in the subject. Pull requests should link related issues, summarize behavior changes, note migrations or env needs, and include before/after screenshots for UI updates. Confirm both `go test ./...` and `npm run test` results in the PR description before requesting review.

## Environment & Configuration Tips
Copy `.env_example` to `.env` and override only the keys you touch; avoid committing secrets. SQLite state is persisted in `data/one-mcp.db`, so remove it if you need a clean slate. When integrating external services, prefer storing credentials in `.env` and referencing them via `config/` structs rather than hardcoding values.
