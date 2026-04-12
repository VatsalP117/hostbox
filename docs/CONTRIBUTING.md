# Contributing to Hostbox

Thank you for your interest in contributing to Hostbox! This guide will help you get started.

## Development Setup

### Prerequisites

- **Go 1.25+** with CGO enabled (for SQLite)
- **Node.js 20+** and npm
- **Docker** (for build pipeline)
- **SQLite 3** development headers

### Getting Started

```bash
# Clone the repository
git clone https://github.com/vatsalpatel/hostbox.git
cd hostbox

# Install Go dependencies
go mod download

# Install frontend dependencies
cd web && npm install && cd ..

# Run the API server
CGO_ENABLED=1 go run ./cmd/api

# In another terminal, run the frontend dev server
cd web && npm run dev
```

Or use Docker Compose for development:

```bash
docker compose -f docker-compose.dev.yml up
```

## Code Structure

```
hostbox/
├── cmd/
│   ├── api/          # API server entrypoint
│   └── cli/          # CLI tool
│       ├── cmd/      # Cobra commands
│       └── internal/ # CLI-specific packages
├── internal/
│   ├── api/          # HTTP layer
│   │   ├── handlers/ # Request handlers
│   │   ├── middleware/# Auth, rate limiting, CORS
│   │   └── routes/   # Route registration
│   ├── config/       # Configuration loading
│   ├── database/     # SQLite connection + migrations
│   ├── dto/          # Data transfer objects + validation
│   ├── errors/       # Error types
│   ├── logger/       # Structured logging
│   ├── models/       # Domain models
│   ├── platform/     # Platform utilities
│   │   ├── detect/   # Framework detection
│   │   ├── docker/   # Docker client
│   │   └── sanitize/ # Input sanitization
│   ├── repository/   # Database repositories
│   ├── services/     # Business logic
│   │   ├── admin/    # Self-update service
│   │   ├── backup/   # Backup + restore
│   │   ├── caddy/    # Caddy integration
│   │   ├── deployment/ # Deployment management
│   │   ├── github/   # GitHub App integration
│   │   ├── notification/ # Webhooks/Slack/Discord
│   │   └── scheduler/# Background jobs
│   ├── util/         # Shared utilities
│   ├── version/      # Build version info
│   └── worker/       # Build executor + worker pool
├── migrations/       # SQL migration files
├── web/              # React SPA frontend
├── docker/           # Dockerfiles
├── scripts/          # Install scripts
└── docs/             # Documentation
```

## Code Style

### Go

- Format with `gofmt`
- Follow standard Go conventions
- Use structured logging (`slog`)
- All exported types and functions should have godoc comments
- Error wrapping: use `fmt.Errorf("context: %w", err)`

### TypeScript

- Format with Prettier
- Lint with ESLint
- Use TypeScript strict mode

### Commit Messages

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat: add custom domain support
fix: handle nil pointer in deployment handler
docs: update self-hosting guide
test: add repository integration tests
refactor: extract build executor from worker pool
```

## Testing

```bash
# Run all Go tests
CGO_ENABLED=1 go test ./... -count=1

# Run with verbose output
CGO_ENABLED=1 go test ./... -v -count=1

# Run specific package tests
CGO_ENABLED=1 go test ./internal/repository/... -v

# Run with race detection
CGO_ENABLED=1 go test -race ./...

# Frontend tests
cd web && npm test
```

## Pull Request Process

1. Fork the repository
2. Create a feature branch (`git checkout -b feat/my-feature`)
3. Make your changes with tests
4. Ensure all tests pass (`go test ./...`)
5. Commit with conventional commit messages
6. Push and open a Pull Request
7. Wait for CI to pass and a review

## Reporting Issues

- Use GitHub Issues
- Include steps to reproduce
- Include relevant logs or error messages
- Mention your OS, Go version, and Docker version
