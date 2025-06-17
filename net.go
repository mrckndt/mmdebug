package main

import (
	"fmt"
	"net"
	"time"
)

// testTCPConnection tests if a TCP connection can be established to the given host and port.
func testTCPConnection(host string, port int, timeout time.Duration) error {
	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", address, err)
	}
	defer conn.Close()
	return nil
}

// testUDPConnection tests if a UDP connection can be established to the given host and port.
func testUDPConnection(host string, port int, timeout time.Duration) error {
	address := fmt.Sprintf("%s:%d", host, port)
	conn, err := net.DialTimeout("udp", address, timeout)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", address, err)
	}
	defer conn.Close()
	return nil
}

// testConnection tests both TCP and UDP connections to the given host and port.
func testConnection(host string, port int, timeout time.Duration) (tcpErr, udpErr error) {
	tcpErr = testTCPConnection(host, port, timeout)
	udpErr = testUDPConnection(host, port, timeout)
	return
}
