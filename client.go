package main

import (
	"fmt"
	"net/rpc"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// getClient connects to the daemon. If autoStart is true, it spawns the daemon if missing.
func getClient(autoStart bool) (*rpc.Client, error) {
	client, err := rpc.Dial("unix", SocketPath)
	if err == nil {
		return client, nil
	}

	if !autoStart {
		return nil, fmt.Errorf("no active session (start one by adding a note)")
	}

	// Start the daemon in background
	cmd := exec.Command(os.Args[0], "daemon")
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true} // Detach from terminal
	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start daemon: %v", err)
	}

	// Wait for socket to appear (max 1 second)
	for i := 0; i < 20; i++ {
		time.Sleep(50 * time.Millisecond)
		client, err = rpc.Dial("unix", SocketPath)
		if err == nil {
			return client, nil
		}
	}
	return nil, fmt.Errorf("timeout waiting for daemon to start")
}
