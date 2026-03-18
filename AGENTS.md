# gitctl - Fast GitHub TUI

## Architecture Overview

gitctl is a fast Terminal UI (TUI) for interacting with GitHub, designed with a client-server architecture for flexibility and performance.

### Components

#### Backend Server (`cmd/gitctl-backend`)
- Written in Go
- Serves a gRPC API for GitHub operations
- Handles GitHub API interactions
- Stateless design for simplicity
- Communicates via Unix domain socket

#### Frontend TUI (`cmd/gitctl`)
- Written in Go using a TUI framework (e.g., bubbletea, tview)
- Rich terminal interface for user interaction
- gRPC client connecting to backend server
- Fast and responsive UI

### Communication

- **Protocol**: gRPC (defined in `proto/gitctl.proto`)
- **Transport**: Unix domain socket (e.g., `/tmp/gitctl.sock`)
- Future: Could support TCP for remote access

### Initial Features

1. **List Repositories**: Display all repositories for a configured GitHub username
   - No authentication required (uses public GitHub API)
   - Configurable username (default: justinsb)

### Project Structure

```
gitctl/
├── CLAUDE.md              # This file
├── proto/
│   └── gitctl.proto       # gRPC service definitions
├── cmd/
│   ├── gitctl-backend/    # Backend server
│   │   └── main.go
│   └── gitctl/            # Frontend TUI
│       └── main.go
├── internal/
│   ├── backend/           # Backend implementation
│   ├── frontend/          # Frontend TUI implementation
│   └── github/            # GitHub API client
└── go.mod
```

### Development Workflow

1. Start backend server: `go run cmd/gitctl-backend/main.go`
2. Start frontend TUI: `go run cmd/gitctl/main.go`
3. The frontend automatically connects to backend via socket

### Future Enhancements

- Authentication support (OAuth, Personal Access Tokens)
- Multiple GitHub operations (issues, PRs, notifications)
- Multiple frontend options (web UI, mobile)
- Configuration file support
- Caching and performance optimizations
