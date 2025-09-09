# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Liwords (Woogles.io) is a web-based crossword board game platform with real-time multiplayer capabilities. The project consists of:

- **Backend API Server**: Go-based API server using Connect RPC (gRPC-compatible)
- **Frontend**: React/TypeScript UI built with RSBuild
- **Socket Server**: Separate Go service for real-time communication (in liwords-socket repo)
- **Game Engine**: Macondo library provides core game logic (in macondo repo)
- **Infrastructure**: PostgreSQL, Redis, NATS messaging, S3 storage

## Key Commands

### Frontend Development (liwords-ui/)
```bash
# Install dependencies
npm install

# Run development server
npm start

# Build production
npm run build

# Run tests
npm test

# Lint code
npm run lint

# Format code
npm run format

# Full pre-commit check
npm run isready
```

### Backend Development
```bash
# Run API server locally
go run cmd/liwords-api/*.go

# Run tests
go test ./...

# Generate code from proto/sql
go generate

# Run migrations up
migrate -database "postgres://postgres:pass@localhost:5432/liwords?sslmode=disable" -path db/migrations up

# Run migrations down
./migrate_down.sh
```

### Docker Development
```bash
# Full stack with Docker
docker compose up

# Services only (for hybrid development)
docker compose -f dc-local-services.yml up

# Register a bot user
./scripts/utilities/register-bot.sh BotUsername
```

## Architecture

### Service Communication
- **API Server** → **Socket Server**: Via NATS pub/sub for real-time events
- **Frontend** → **API Server**: Connect RPC over HTTP
- **Frontend** → **Socket Server**: WebSocket for real-time updates
- **API Server** → **PostgreSQL**: Primary data store
- **API Server** → **Redis**: Session storage, presence, chat history

### Key Patterns

1. **Code Generation**:
   - Proto files → Go/TypeScript code via `buf generate`
   - SQL queries → Go code via `sqlc generate`
   - Run `go generate` after modifying .proto or .sql files

2. **Service Structure**:
   - Each domain has a service in `pkg/` (e.g., `pkg/gameplay`, `pkg/tournament`)
   - Services expose Connect RPC handlers
   - Database access through generated sqlc code in `pkg/stores/`

3. **Real-time Events**:
   - Game events flow through NATS
   - Socket server broadcasts to connected clients
   - Event types defined in `api/proto/ipc/`

4. **Authentication**:
   - JWT tokens for API authentication
   - Session cookies for web clients
   - Bot accounts have `internal_bot` flag

### Important Directories

- `api/proto/`: Protocol buffer definitions
- `cmd/`: Entry points for various services
- `pkg/`: Core business logic and services
- `db/migrations/`: PostgreSQL schema migrations
- `db/queries/`: SQL queries for sqlc
- `liwords-ui/src/`: Frontend React code
- `rpc/`: Generated RPC code

## Testing

### Running Tests
```bash
# Backend unit tests
go test ./pkg/...

# Frontend tests
cd liwords-ui && npm test

# Integration tests (requires running services)
go test ./pkg/integration_testing/...
```

### Test Patterns
- Go tests use standard `testing` package
- Frontend uses Vitest
- Test data in `testdata/` directories
- Golden files for snapshot testing

## Common Development Tasks

### Adding a New RPC Endpoint
1. Define the service method in `api/proto/[service]/[service].proto`
2. Run `go generate` to generate code
3. Implement the handler in `pkg/[service]/service.go`
4. Add the service to the router in `cmd/liwords-api/main.go`

### Adding a Database Query
1. Write the SQL query in `db/queries/[domain].sql`
2. Run `go generate` to generate the Go code
3. Use the generated methods in your service

### Modifying the Database Schema
1. Create a new migration: `./gen_migration.sh [migration_name]`
2. Write the up/down SQL in `db/migrations/`
3. Run migrations: `migrate -database "..." -path db/migrations up`

## Environment Variables

Key environment variables (see docker-compose.yml for full list):
- `DB_*`: PostgreSQL connection settings
- `REDIS_URL`: Redis connection
- `NATS_URL`: NATS server URL
- `SECRET_KEY`: JWT signing key
- `MACONDO_DATA_PATH`: Path to game data files
- `AWS_*`: S3 configuration for uploads

## Debugging Tips

- Enable debug logging: `DEBUG=1`
- Access pprof: http://localhost:8001/debug/pprof/
- NATS monitoring: Connect to NATS and subscribe to `>` for all messages
- Database queries are logged when `DEBUG=1`