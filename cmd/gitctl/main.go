package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/justinsb/gitctl/internal/frontend"
)

var (
	socketPath = flag.String("socket", "/tmp/gitctl.sock", "Unix domain socket path")
	username   = flag.String("username", "justinsb", "GitHub username to query")
)

func main() {
	flag.Parse()

	// Build an HTTP client that communicates with the backend over the Unix socket.
	client := frontend.NewClient(*socketPath)

	// Create TUI program
	p := tea.NewProgram(
		frontend.NewModel(client, *username),
		tea.WithAltScreen(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Printf("Error running program: %v\n", err)
		os.Exit(1)
	}
}
