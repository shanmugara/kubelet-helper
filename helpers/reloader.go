package helpers

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/fsnotify/fsnotify"
)

const (
	debounceDelay = 2 * time.Second
)

var log = logrus.New()

// WatchKubeletConfig watches the kubelet config directory and reloads kubelet on changes
func WatchKubeletConfig(kubeletConfigPath string) error {
	// Create fsnotify watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}
	defer watcher.Close()

	// Ensure the config directory exists
	if _, err := os.Stat(kubeletConfigPath); os.IsNotExist(err) {
		return fmt.Errorf("config directory %s does not exist", kubeletConfigPath)
	}

	// Add the directory to the watcher
	err = watcher.Add(kubeletConfigPath)
	if err != nil {
		return fmt.Errorf("failed to watch directory %s: %w", kubeletConfigPath, err)
	}

	log.Printf("Started watching %s for changes...", kubeletConfigPath)

	// Debounce timer to avoid multiple reloads for rapid changes
	var debounceTimer *time.Timer

	// Watch for events
	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return fmt.Errorf("watcher event channel closed")
			}

			// Only process write and create events
			if event.Op&fsnotify.Write == fsnotify.Write || event.Op&fsnotify.Create == fsnotify.Create {
				log.Printf("Detected change: %s (%s)", event.Name, event.Op)
				if !strings.HasSuffix(event.Name, ".conf") {
					log.Printf("Ignoring non-config file change: %s", event.Name)
					continue
				}

				// Reset debounce timer
				if debounceTimer != nil {
					debounceTimer.Stop()
				}

				debounceTimer = time.AfterFunc(debounceDelay, func() {
					if err := reloadKubelet(); err != nil {
						log.Printf("Error reloading kubelet: %v", err)
					} else {
						log.Printf("Successfully reloaded kubelet after detecting changes in %s", filepath.Base(event.Name))
					}
				})
			}

		case err, ok := <-watcher.Errors:
			if !ok {
				return fmt.Errorf("watcher error channel closed")
			}
			log.Printf("Watcher error: %v", err)
		}
	}
}

// reloadKubelet reloads the kubelet service without restarting it
func reloadKubelet() error {
	log.Println("Reloading kubelet...")

	// Try systemctl reload first
	cmd := exec.Command("systemctl", "daemon-reload")
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("systemctl daemon-reload failed: %v, output: %s", err, string(output))
	}
	cmd = exec.Command("systemctl", "restart", "kubelet")
	output, err = cmd.CombinedOutput()

	if err != nil {
		log.Printf("systemctl reload failed: %v, output: %s", err, string(output))
		log.Println("Attempting alternative reload method using SIGHUP...")

		// Alternative: send SIGHUP signal to kubelet process
		cmd = exec.Command("pkill", "-HUP", "-f", "^/usr/bin/kubelet")
		output, err = cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("failed to reload kubelet with SIGHUP: %w, output: %s", err, string(output))
		}
	}

	log.Println("Kubelet reload command executed successfully")
	return nil
}
