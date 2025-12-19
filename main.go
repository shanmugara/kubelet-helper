package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/shanmugara/kubelet-helper/helpers"
)

func main() {
	kubeletConfigPath := flag.String("config-path", "/var/lib/kubelet/conf.d", "Path to kubelet config directory to watch")
	flag.Parse()

	log.Println("Kubelet Config Reloader starting...")
	log.Printf("Watching config path: %s", *kubeletConfigPath)

	// Handle graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start watching in a goroutine
	errChan := make(chan error, 1)
	go func() {
		if err := helpers.WatchKubeletConfig(*kubeletConfigPath); err != nil {
			errChan <- err
		}
	}()

	// Wait for shutdown signal or error
	select {
	case sig := <-sigChan:
		log.Printf("Received signal %v, shutting down gracefully...", sig)
	case err := <-errChan:
		log.Fatalf("Fatal error: %v", err)
	}

	log.Println("Kubelet Config Reloader stopped")
}
