# gitctl - Fast GitHub TUI

## Architecture Overview

gitctl is a fast Terminal UI (TUI) for interacting with GitHub, designed with a client-server architecture following Kubernetes design patterns.

### Design Principles

This project follows several Kubernetes design patterns:

- **CRDs (Custom Resource Definitions)**: Resources like `GitRepo` are modeled as Kubernetes-style typed objects with `apiVersion`, `kind`, `metadata`, `spec`, and `status` fields.
- **Wire Protocol**: The API uses the Kubernetes wire protocol — JSON over REST with standard conventions for list/get endpoints.
- **Controllers**: Background controllers poll external sources (e.g. GitHub API) and reconcile state into storage, rather than fetching on demand. This decouples API serving from data collection.
- **Storage-backed serving**: The API server reads from storage rather than making live calls to external APIs. This provides fast, consistent responses and resilience to upstream outages.

### Components

#### Backend Server (`cmd/gitctl-backend`)
- Written in Go
- Serves a Kubernetes-style REST API (JSON over HTTP)
- Reads from storage to serve requests
- Communicates via Unix domain socket

#### Controllers (`internal/controller`)
- **GitRepoController**: Polls GitHub for repositories on a configurable interval and writes them to storage. Runs as a background goroutine in the backend process.

#### Storage (`internal/storage`)
- Interface-based design (`storage.GitRepoStore`) allowing pluggable backends
- **memorystorage**: In-memory implementation for development and simple deployments
- Future: could add persistent backends (SQLite, etcd, etc.)

#### Frontend TUI (`cmd/gitctl`)
- Written in Go using Bubble Tea
- Rich terminal interface for user interaction
- HTTP client connecting to backend server via Unix domain socket

### Communication

- **Protocol**: Kubernetes-style JSON over REST
- **Transport**: Unix domain socket (default: `/tmp/gitctl.sock`)
- Future: Could support TCP for remote access

### Data Flow

```
GitHub API  --(poll)-->  Controller  --(write)-->  Storage  --(read)-->  API Server  --(HTTP)-->  Frontend TUI
```

### Initial Features

1. **List Repositories**: Display all repositories for a configured GitHub username
   - No authentication required (uses public GitHub API)
   - Configurable username (default: justinsb)

### Project Structure

```
gitctl/
├── CLAUDE.md              # Project instructions
├── AGENTS.md              # This file — architecture overview
├── cmd/
│   ├── gitctl-backend/    # Backend server
│   │   └── main.go
│   └── gitctl/            # Frontend TUI
│       └── main.go
├── internal/
│   ├── api/               # Kubernetes CRD-style type definitions
│   ├── backend/           # API server (reads from storage)
│   ├── controller/        # Controllers (poll external sources, write to storage)
│   ├── frontend/          # Frontend TUI implementation
│   ├── github/            # GitHub API client
│   └── storage/           # Storage interfaces and implementations
│       ├── interfaces.go
│       └── memorystorage/
│           └── memorystorage.go
└── go.mod
```

### Development Workflow

1. Start backend server: `go run cmd/gitctl-backend/main.go`
   - Flags: `-username` (default: justinsb), `-sync-interval` (default: 5m), `-socket` (default: /tmp/gitctl.sock)
2. Start frontend TUI: `go run cmd/gitctl/main.go`
3. The frontend automatically connects to backend via socket
4. The backend controller syncs repos from GitHub immediately on startup, then at the configured interval

### Future Enhancements

- Authentication support (OAuth, Personal Access Tokens)
- Watch/informer pattern for real-time updates
- Multiple GitHub operations (issues, PRs, notifications)
- Persistent storage backends (SQLite, etcd)
- Multiple frontend options (web UI, mobile)
- Configuration file support
