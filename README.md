# gospin

> Spin up production-ready Go backends in seconds.

`gospin` is a zero-dependency Go **library and CLI** that generates a complete, opinionated Go backend project — so you skip the boilerplate and start writing real code immediately. Use the `gospin` command from your terminal, or import the `github.com/nidhi-2619/gospin` package and call `Generate` from your own tooling.

## What gets generated

```
myapp/
├── cmd/
│   ├── api/main.go          # HTTP server entry point (wired & ready)
│   └── migrate/main.go      # Database migration runner
├── bootstrap/               # Dependency wiring / composition root
├── api/health/              # Working /health endpoint + unit tests
├── server/                  # Gin engine, middleware, graceful shutdown
├── config/                  # Typed env config (DB, HTTP, JWT, Redis)
├── common/                  # JSON response helpers, shared utilities
├── infra/
│   ├── db/                  # PostgreSQL connection pool
│   ├── cache/               # Redis client (stub, ready to wire)
│   └── jwt/                 # JWT sign/verify manager (stub)
├── middleware/               # Auth middleware (Bearer token stub)
├── model/                   # Domain entity structs
├── repository/              # Database query layer
├── utils/                   # Helper functions
├── constants/               # Immutable app constants
├── mocks/                   # Test mock implementations
├── Dockerfile               # Multi-stage production build
├── docker-compose.yml       # App + PostgreSQL, ready to run
├── Makefile                 # run, build, test, lint, migrate, docker
├── .env.example             # All env vars documented
└── README.md
```

## Stack

| Layer | Technology |
|---|---|
| Framework | [Gin](https://github.com/gin-gonic/gin) |
| Database | PostgreSQL via `database/sql` + `lib/pq` |
| Logging | [Uber Zap](https://github.com/uber-go/zap) |
| Language | Go 1.22+ |

## Installation

### Option 1 — go install (recommended)

```bash
go install github.com/YOURUSERNAME/gospin@latest
```

### Option 2 — Download binary

Grab the latest binary for your platform from the [Releases](https://github.com/YOURUSERNAME/gospin/releases) page.

```bash
chmod +x gospin
sudo mv gospin /usr/local/bin/
```

### Option 3 — Build from source

```bash
git clone https://github.com/YOURUSERNAME/gospin.git
cd gospin
go build -o gospin .
```

## Usage

```bash
# Basic — creates ./myapp
gospin github.com/yourorg/myapp

# Custom output directory
gospin -module github.com/yourorg/myapp -output ./services/myapp

# Skip go mod init / tidy (useful in CI or offline)
gospin --skip-init github.com/yourorg/myapp

# Show version
gospin --version
```

## Quick start with a generated project

```bash
gospin github.com/yourorg/myapp
cd myapp

cp .env.example .env        # edit DB credentials
docker compose up -d        # start PostgreSQL
go mod tidy                 # download dependencies
go run ./cmd/api            # start the server

curl localhost:8080/health
# {"status":"UP","uptime":"1.2s","version":"1.0.0"}
```

## Architecture — Layered Root Layout

`gospin` generates a **Layered Root Layout** — all core packages live at the project root with no `internal/` wrapper. Each layer has a single responsibility:

```
cmd        →  entry points only (no business logic)
bootstrap  →  wires all dependencies together (composition root)
api        →  HTTP handlers (thin, delegate to services)
server     →  Gin engine configuration and lifecycle
config     →  reads environment, returns typed structs
infra      →  infrastructure connections (DB, cache, JWT)
middleware →  reusable Gin middleware
model      →  domain structs / entities
repository →  database queries
```

## Flags

| Flag | Default | Description |
|---|---|---|
| `-module` | required | Go module path (e.g. `github.com/org/project`) |
| `-output` | last segment of module | Output directory |
| `--skip-init` | false | Skip `go mod init` and `go mod tidy` |
| `--version` | — | Print version and exit |

## Contributing

Pull requests are welcome! To add a new template or flag:

1. Fork the repo
2. Create a feature branch (`git checkout -b feat/my-feature`)
3. Make your changes in `main.go`
4. Test: `go run . --skip-init github.com/test/project`
5. Open a PR

## License

MIT — see [LICENSE](LICENSE)
