package testutil

import (
	"fmt"
	"net"
)

// GetFreePort asks the kernel for a free open port that is ready to use.
// This is a common pattern in integration tests to avoid port conflicts.
func GetFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, fmt.Errorf("failed to resolve TCP address: %w", err)
	}

	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, fmt.Errorf("failed to listen on TCP address: %w", err)
	}
	defer listener.Close()

	return listener.Addr().(*net.TCPAddr).Port, nil
}

// GetFreeAddress returns a free address in the format "localhost:port"
func GetFreeAddress() (string, error) {
	port, err := GetFreePort()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("localhost:%d", port), nil
}
