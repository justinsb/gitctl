package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/justinsb/gitctl/internal/backend"
	pb "github.com/justinsb/gitctl/proto"
	"google.golang.org/grpc"
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

	// Make socket accessible
	if err := os.Chmod(*socketPath, 0666); err != nil {
		log.Fatalf("Failed to chmod socket: %v", err)
	}

	// Create gRPC server
	grpcServer := grpc.NewServer()

	// Register GitCtl service
	pb.RegisterGitCtlServer(grpcServer, backend.NewServer())

	// Handle graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		fmt.Println("\nShutting down server...")
		grpcServer.GracefulStop()
	}()

	log.Printf("Backend server listening on %s", *socketPath)
	if err := grpcServer.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
