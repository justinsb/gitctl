# gitctl - Fast GitHub TUI

A fast Terminal UI for interacting with GitHub, built with Go and gRPC.

## Architecture

- **Backend**: gRPC server that handles GitHub API interactions
- **Frontend**: Terminal UI built with [Bubble Tea](https://github.com/charmbracelet/bubbletea)
- **Communication**: gRPC over Unix domain socket

See [CLAUDE.md](CLAUDE.md) for detailed architecture documentation.

## Prerequisites

- Go 1.21 or later
- Protocol Buffers compiler (protoc)

## Building

```bash
# Install dependencies
go mod download

# Generate protobuf code (if needed)
make proto

# Build both binaries
make build
```

## Running

### Terminal 1 - Start the backend server

```bash
go run cmd/gitctl-backend/main.go
```

Or with custom socket path:

```bash
go run cmd/gitctl-backend/main.go -socket /tmp/my-gitctl.sock
```

### Terminal 2 - Start the frontend TUI

```bash
go run cmd/gitctl/main.go
```

Or with custom options:

```bash
go run cmd/gitctl/main.go -username octocat -socket /tmp/my-gitctl.sock
```

## Usage

The TUI displays repositories for the configured GitHub username (default: `justinsb`).

### Keyboard shortcuts:

- `↑/↓` or `j/k` - Navigate through repositories
- `/` - Filter repositories
- `q` or `Ctrl+C` - Quit

## Development

### Project Structure

```
gitctl/
├── cmd/
│   ├── gitctl-backend/   # Backend gRPC server
│   └── gitctl/           # Frontend TUI
├── internal/
│   ├── backend/          # Backend server implementation
│   ├── frontend/         # TUI model and views
│   └── github/           # GitHub API client
├── proto/
│   ├── gitctl.proto      # gRPC service definition
│   ├── gitctl.pb.go      # Generated protobuf code
│   └── gitctl_grpc.pb.go # Generated gRPC code
├── CLAUDE.md             # Architecture documentation
└── README.md             # This file
```

### Regenerating Protobuf Code

If you modify `proto/gitctl.proto`, regenerate the code:

```bash
make proto
```

Or manually:

```bash
protoc --go_out=. --go_opt=paths=source_relative \
    --go-grpc_out=. --go-grpc_opt=paths=source_relative \
    proto/gitctl.proto
```

## Features

### Current

- List all public repositories for a GitHub user
- Fast, responsive TUI
- Filter/search repositories

### Planned

- Authentication (OAuth, Personal Access Tokens)
- View issues and pull requests
- Repository details view
- Notifications
- Multiple backend connections
