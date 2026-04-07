package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/justinsb/gitctl/internal/api"
	"github.com/justinsb/gitctl/internal/backend"
	"github.com/justinsb/gitctl/internal/controller"
	"github.com/justinsb/gitctl/internal/github"
	"github.com/justinsb/gitctl/internal/storage"
)

var (
	socketPath   = flag.String("socket", "/tmp/gitctl.sock", "Unix domain socket path")
	tcpAddr      = flag.String("tcp", "127.0.0.1:8484", "TCP address to listen on (empty to disable)")
	username     = flag.String("username", "justinsb", "GitHub username to sync repositories for")
	syncInterval = flag.Duration("sync-interval", 5*time.Minute, "How often to poll GitHub for repository updates")
	dataDir      = flag.String("data-dir", "", "Directory for persistent data (views, etc.); defaults to ~/.config/gitctl")
)

// defaultDataDir returns the platform default data directory for gitctl.
func defaultDataDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return home + "/.config/gitctl", nil
}

func main() {
	ctx := context.Background()
	if err := Run(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func Run(ctx context.Context) error {
	flag.Parse()

	dir := *dataDir
	if dir == "" {
		var err error
		dir, err = defaultDataDir()
		if err != nil {
			return err
		}
	}

	// Remove existing socket if it exists
	if err := os.RemoveAll(*socketPath); err != nil {
		return fmt.Errorf("failed to remove existing socket: %w", err)
	}

	// Create Unix domain socket listener
	listener, err := net.Listen("unix", *socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", *socketPath, err)
	}
	defer os.Remove(*socketPath)

	// Make socket accessible to all users
	if err := os.Chmod(*socketPath, 0666); err != nil {
		return fmt.Errorf("failed to chmod socket: %w", err)
	}

	// Set up per-resource stores.
	repoStore := storage.NewResourceStore[api.GitRepo]()
	prStore := storage.NewResourceStore[api.PullRequest]()
	issueStore := storage.NewResourceStore[api.Issue]()
	commentStore := storage.NewResourceStore[api.Comment]()
	commitStore := storage.NewResourceStore[api.PRCommit]()
	checkRunStore := storage.NewResourceStore[api.CheckRun]()
	prFileStore := storage.NewResourceStore[api.PRFile]()
	reviewCommentStore := storage.NewResourceStore[api.ReviewComment]()

	// Set up persistent view store. Create the data directory if needed.
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory %s: %w", dir, err)
	}
	viewsFile := dir + "/views.json"
	viewStore, err := storage.NewFileStorage[api.View](
		func(v api.View) string { return v.Metadata.Name },
		viewsFile,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize view store: %w", err)
	}
	fmt.Fprintf(os.Stderr, "View store initialized from %s\n", viewsFile)

	githubClient := github.NewClient()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Start the controllers to poll GitHub and populate storage.
	repoCtrl := controller.NewGitRepoController(githubClient, repoStore, *username, *syncInterval)
	go repoCtrl.Run(ctx)

	prCtrl := controller.NewPullRequestController(githubClient, prStore, commentStore, *username, *syncInterval)
	go prCtrl.Run(ctx)

	issueCtrl := controller.NewIssueController(githubClient, issueStore, commentStore, *username, *syncInterval)
	go issueCtrl.Run(ctx)

	// Create the API handler that reads from storage.
	handler := backend.NewServer(repoStore, prStore, issueStore, commentStore, commitStore, checkRunStore, prFileStore, reviewCommentStore, viewStore, githubClient)

	// Start Unix socket server.
	unixServer := &http.Server{Handler: handler}
	go func() {
		fmt.Fprintf(os.Stderr, "Backend server listening on %s\n", *socketPath)
		if err := unixServer.Serve(listener); err != nil && err != http.ErrServerClosed {
			fmt.Fprintf(os.Stderr, "Failed to serve on Unix socket: %v\n", err)
		}
	}()

	// Optionally start TCP server (for non-Unix-socket clients like the macOS app).
	var tcpServer *http.Server
	if *tcpAddr != "" {
		tcpListener, err := net.Listen("tcp", *tcpAddr)
		if err != nil {
			return fmt.Errorf("failed to listen on %s: %w", *tcpAddr, err)
		}
		tcpServer = &http.Server{Handler: handler}
		go func() {
			fmt.Fprintf(os.Stderr, "Backend server listening on %s\n", *tcpAddr)
			if err := tcpServer.Serve(tcpListener); err != nil && err != http.ErrServerClosed {
				fmt.Fprintf(os.Stderr, "Failed to serve on TCP: %v\n", err)
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
		fmt.Fprintf(os.Stderr, "Error closing Unix server: %v\n", err)
	}
	if tcpServer != nil {
		if err := tcpServer.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error closing TCP server: %v\n", err)
		}
	}
	return nil
}
