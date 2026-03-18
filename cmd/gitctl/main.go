package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/justinsb/gitctl/internal/frontend"
	pb "github.com/justinsb/gitctl/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	socketPath = flag.String("socket", "/tmp/gitctl.sock", "Unix domain socket path")
	username   = flag.String("username", "justinsb", "GitHub username to query")
)

func main() {
	flag.Parse()

	// Connect to backend via Unix domain socket
	conn, err := grpc.Dial(
		"unix://"+*socketPath,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		log.Fatalf("Failed to connect to backend: %v", err)
	}
	defer conn.Close()

	client := pb.NewGitCtlClient(conn)

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
