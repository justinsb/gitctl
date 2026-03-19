package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/justinsb/gitctl/internal/backend"
)

var (
	socketPath = flag.String("socket", "/tmp/gitctl.sock", "Unix domain socket path")
)

func main() {
	flag.Parse()

	// Remove existing socket if it exists
	if err := os.RemoveAll(*socketPath); err != nil {
		log.Fatalf("Failed to remove existing socket: %v", err)
	}

	// Create Unix domain socket listener
	listener, err := net.Listen("unix", *socketPath)
	if err != nil {
		log.Fatalf("Failed to listen on %s: %v", *socketPath, err)
	}
	defer os.Remove(*socketPath)

	// Make socket accessible to all users
	if err := os.Chmod(*socketPath, 0666); err != nil {
		log.Fatalf("Failed to chmod socket: %v", err)
	}

	server := &http.Server{
		Handler: backend.NewServer(),
	}

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\nShutting down server...")
		if err := server.Close(); err != nil {
			log.Printf("Error closing server: %v", err)
		}
	}()

	log.Printf("Backend server listening on %s", *socketPath)
	if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to serve: %v", err)
	}
}
