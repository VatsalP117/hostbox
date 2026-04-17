# Hostbox

**Self-hostable deployment platform.** Push code, get a URL. Like Vercel, but on your own server.

[![Test](https://github.com/vatsalpatel/hostbox/actions/workflows/test.yml/badge.svg)](https://github.com/vatsalpatel/hostbox/actions/workflows/test.yml)
[![Go Report](https://goreportcard.com/badge/github.com/vatsalpatel/hostbox)](https://goreportcard.com/report/github.com/vatsalpatel/hostbox)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

---

## Features

- 🚀 **Git-push deployments** — push to GitHub, get a live preview URL automatically
- 🔒 **Built-in SSL** — automatic HTTPS with Let's Encrypt via Caddy
- 🌐 **Custom domains** — add your own domains with automatic certificate provisioning
- 🔀 **Preview deployments** — every branch and PR gets its own URL
- 📦 **Framework detection** — auto-detects Next.js, Vite, Astro, Remix, Hugo, and more
- 🐳 **Docker-based builds** — isolated, reproducible builds in containers
- 🔑 **Encrypted environment variables** — AES-256-GCM encryption at rest
- 📊 **Real-time build logs** — stream build output via SSE
- ↩️ **Instant rollbacks** — one-click rollback to any previous deployment
- 🛡️ **Admin dashboard** — user management, system stats, settings
- 💻 **CLI tool** — deploy, manage projects, and configure from your terminal
- 📱 **Responsive dashboard** — React SPA with dark mode support

## Quick Start

### One-Line Install

```bash
curl -fsSL https://raw.githubusercontent.com/vatsalpatel/hostbox/main/scripts/install.sh | sudo bash
```

This will:
1. Install Docker (if needed)
2. Prompt for your domain and email
3. Generate secrets and start Hostbox

### Manual Install

```bash
# Clone and configure
git clone https://github.com/vatsalpatel/hostbox.git /opt/hostbox
cd /opt/hostbox
cp .env.production.example .env
# Edit .env with your domain, email, and secrets

# Start
docker compose up -d
```

### Access Dashboard

Open `https://your-domain.com` and create your admin account.

## Architecture

Hostbox runs as a **single Go binary** alongside **Caddy** as a reverse proxy:

```
Internet → Caddy (SSL/routing) → Hostbox API (Go)
                                     ├── SQLite (WAL mode)
                                     ├── Docker (builds)
                                     └── Static files (deployments)
```

- **Single binary**: API server, build worker, and dashboard in one executable
- **SQLite**: Zero-config database with WAL mode for concurrent reads
- **Caddy**: Automatic SSL, HTTP/3, and dynamic route configuration via Admin API
- **Docker**: Isolated builds with resource limits and security hardening

See [ARCHITECTURE.md](docs/ARCHITECTURE.md) for the full technical design.

## CLI

Install the CLI:

```bash
curl -fsSL https://raw.githubusercontent.com/vatsalpatel/hostbox/main/scripts/install-cli.sh | bash
```

Common commands:

```bash
hostbox login                    # Authenticate with your server
hostbox projects                 # List projects
hostbox link                     # Link current directory to a project
hostbox deploy                   # Trigger a deployment
hostbox status                   # View deployment status
hostbox logs <deployment-id>     # Stream build logs
hostbox domains add example.com  # Add a custom domain
hostbox env set API_KEY=secret   # Set environment variable
```

## Self-Hosting

See [docs/SELF-HOSTING.md](docs/SELF-HOSTING.md) for detailed self-hosting instructions including:
- Server requirements
- DNS setup
- GitHub App configuration
- Backup and restore
- Troubleshooting

## Development

```bash
# Prerequisites: Go 1.25+, Node 20+, Docker, SQLite

# Backend
go run ./cmd/api

# Frontend
cd web && npm install && npm run dev

# Or use Docker Compose
docker compose -f docker-compose.dev.yml up
```

See [docs/ONBOARDING.md](docs/ONBOARDING.md) for an implementation-first contributor guide that explains the current codebase, local workflow, and known gaps.

See [docs/CONTRIBUTING.md](docs/CONTRIBUTING.md) for the full development guide.

## Tech Stack

| Component | Technology |
|-----------|-----------|
| API Server | Go, Echo framework |
| Database | SQLite (WAL mode) |
| Reverse Proxy | Caddy |
| Builds | Docker |
| Frontend | React, TypeScript, Tailwind CSS |
| CLI | Go, Cobra |

## License

MIT — see [LICENSE](LICENSE) for details.
