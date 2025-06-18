package main

import (
	"fmt"
	"net"
	"time"

	"github.com/jedib0t/go-pretty/v6/text"
)


// testTCPConnection tests if a TCP connection can be established to the given host and port.
// It returns an error if the connection fails within the specified timeout duration.
// The connection is automatically closed after successful establishment.
func testTCPConnection(host string, port int, timeout time.Duration) error {
	address := fmt.Sprintf("%s:%d", host, port)
	start := time.Now()

	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		duration := time.Since(start)
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return fmt.Errorf("TCP connection to %s timed out after %v: %w", address, duration, err)
		}
		return fmt.Errorf("TCP connection to %s failed after %v: %w", address, duration, err)
	}
	defer conn.Close()
	return nil
}

// printTCPResult prints a colorized TCP test result
func printTCPResult(host string, port int, err error) {
	if err != nil {
		fmt.Printf("%s\n", text.Colors{text.Bold, text.FgRed}.Sprintf("TCP connection to %s:%d failed", host, port))
	} else {
		fmt.Printf("%s\n", text.Colors{text.Bold, text.FgGreen}.Sprintf("TCP connection to %s:%d successful", host, port))
	}
}


