package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/justinsb/gitctl/internal/backend"
	"github.com/justinsb/gitctl/internal/controller"
	"github.com/justinsb/gitctl/internal/github"
	"github.com/justinsb/gitctl/internal/storage/memorystorage"
)

var (
	socketPath   = flag.String("socket", "/tmp/gitctl.sock", "Unix domain socket path")
	tcpAddr      = flag.String("tcp", "127.0.0.1:8484", "TCP address to listen on (empty to disable)")
	username     = flag.String("username", "justinsb", "GitHub username to sync repositories for")
	syncInterval = flag.Duration("sync-interval", 5*time.Minute, "How often to poll GitHub for repository updates")
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

	// Set up storage and controllers.
	store := memorystorage.New()
	githubClient := github.NewClient()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start the controllers to poll GitHub and populate storage.
	repoCtrl := controller.NewGitRepoController(githubClient, store, *username, *syncInterval)
	go repoCtrl.Run(ctx)

	prCtrl := controller.NewPullRequestController(githubClient, store, store, *username, *syncInterval)
	go prCtrl.Run(ctx)

	issueCtrl := controller.NewIssueController(githubClient, store, store, *username, *syncInterval)
	go issueCtrl.Run(ctx)

	// Create the API handler that reads from storage.
	handler := backend.NewServer(store, store, store, store, githubClient)

	// Start Unix socket server.
	unixServer := &http.Server{Handler: handler}
	go func() {
		log.Printf("Backend server listening on %s", *socketPath)
		if err := unixServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to serve on Unix socket: %v", err)
		}
	}()

	// Optionally start TCP server (for non-Unix-socket clients like the macOS app).
	var tcpServer *http.Server
	if *tcpAddr != "" {
		tcpListener, err := net.Listen("tcp", *tcpAddr)
		if err != nil {
			log.Fatalf("Failed to listen on %s: %v", *tcpAddr, err)
		}
		tcpServer = &http.Server{Handler: handler}
		go func() {
			log.Printf("Backend server listening on %s", *tcpAddr)
			if err := tcpServer.Serve(tcpListener); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Failed to serve on TCP: %v", err)
			}
		}()
	}

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	<-sigCh
	fmt.Println("\nShutting down server...")
	cancel()
	if err := unixServer.Close(); err != nil {
		log.Printf("Error closing Unix server: %v", err)
	}
	if tcpServer != nil {
		if err := tcpServer.Close(); err != nil {
			log.Printf("Error closing TCP server: %v", err)
		}
	}
}
