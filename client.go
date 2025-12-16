package main

import (
	"fmt"
	"net/rpc"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// getClient attempts to connect to the running daemon via Unix Socket.
// if autoStart is true, it spawns the daemon process if it isn't running.
func getClient(autoStart bool) (*rpc.Client, error) {
	// 1. Try to connect immediately
	client, err := rpc.Dial("unix", SocketPath)
	if err == nil {
		return client, nil
	}

	// 2. If connection failed and we shouldn't auto-start (e.g., 'list' command), fail.
	if !autoStart {
		return nil, fmt.Errorf("no active session. Start one with 'cnote add'")
	}

	// 3. Spawn the Daemon
	// We call the same binary with the hidden "daemon" command.
	cmd := exec.Command(os.Args[0], "daemon")

	// Setsid: true is critical. It detaches the child process from this terminal.
	// If we don't do this, closing the terminal kills the daemon.
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}

	err = cmd.Start()
	if err != nil {
		return nil, fmt.Errorf("failed to start daemon: %v", err)
	}

	// 4. Wait loop: Wait for the socket file to appear (max 1 second)
	for i := 0; i < 20; i++ {
		time.Sleep(50 * time.Millisecond)
		client, err = rpc.Dial("unix", SocketPath)
		if err == nil {
			return client, nil
		}
	}
	return nil, fmt.Errorf("timeout waiting for daemon to start")
}
