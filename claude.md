# Claude Code Developer Notes for Liwords

This document contains project-specific conventions and commands for working with the Liwords codebase.

## Protocol Buffers

To regenerate protocol buffer files after making changes to `.proto` files:

```bash
go generate
```

**Do not use** `make proto` - use `go generate` instead.

## Project Structure

- `/api/proto/` - Protocol buffer definitions
- `/liwords-ui/` - React TypeScript frontend
- `/pkg/` - Go backend packages

## Common Commands

### Backend (Go)
- Build: `go build ./pkg/...` (or `go build ./cmd/...` for binaries)
- Test: `source local.env && go test ./pkg/...` (must source local.env first!)
- Generate protos: `go generate`
- Manage dependencies: `go mod tidy`

### Running Tests

**IMPORTANT**: Many tests require environment variables from `local.env`. Always source it first:

```bash
source local.env && go test ./pkg/league/... -v
```

Some tests also require a running PostgreSQL database. The test database connection uses:
- `TEST_DB_HOST` and `TEST_DB_NAME` for test-specific database
- `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD` for connection details

### Fixing Broken Tests

When tests don't compile, common issues include:

1. **Mock interfaces missing methods**: If a mock doesn't implement an interface, check the interface definition in `pkg/stores/*/db.go` and add the missing method stubs to the mock.

2. **Function signatures changed**: If test calls have wrong argument types, check the actual function signature and update the test calls. For example, if a function now takes a struct instead of individual arguments, update accordingly.

3. **Compile test first**: Use `go test -c ./pkg/path/...` to compile without running - this catches compile errors quickly.

### Frontend (React/TypeScript)
- Build: `npm run build` (from liwords-ui directory)
- Format: `npx prettier --write <files>`
- Type check: `npx tsc`

## Development Notes

- The project uses React Router v7 (import from `react-router`, not `react-router-dom`)
- Theme-aware styling uses SCSS with `@include colorModed()` mixin
- Tournament URLs include prefix: `/tournament/:slug` or `/club/:slug`
