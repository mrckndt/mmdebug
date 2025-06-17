package main

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"net"
	"time"
)

// tlsTestResult contains information about a TLS handshake test.
type tlsTestResult struct {
	success          bool
	version          uint16
	cipherSuite      uint16
	serverName       string
	peerCertificates int
	err              error
}

// testTLSHandshake performs a TLS handshake similar to openssl s_client.
func testTLSHandshake(host string, port int, timeout time.Duration) *tlsTestResult {
	result := &tlsTestResult{
		serverName: host,
	}

	address := fmt.Sprintf("%s:%d", host, port)

	// Create TLS configuration
	config := &tls.Config{
		ServerName: host,
	}

	// Establish connection with timeout
	dialer := &net.Dialer{
		Timeout: timeout,
	}

	conn, err := tls.DialWithDialer(dialer, "tcp", address, config)
	if err != nil {
		result.err = fmt.Errorf("TLS handshake failed: %w", err)
		return result
	}
	defer conn.Close()

	// Get connection state
	state := conn.ConnectionState()

	result.success = true
	result.version = state.Version
	result.cipherSuite = state.CipherSuite
	result.peerCertificates = len(state.PeerCertificates)

	return result
}

// testTLSHandshakeInsecure performs a TLS handshake without certificate verification.
func testTLSHandshakeInsecure(host string, port int, timeout time.Duration) *tlsTestResult {
	result := &tlsTestResult{
		serverName: host,
	}

	address := fmt.Sprintf("%s:%d", host, port)

	// Create TLS configuration with insecure verification
	config := &tls.Config{
		ServerName:         host,
		InsecureSkipVerify: true,
	}

	// Establish connection with timeout
	dialer := &net.Dialer{
		Timeout: timeout,
	}

	conn, err := tls.DialWithDialer(dialer, "tcp", address, config)
	if err != nil {
		result.err = fmt.Errorf("TLS handshake failed: %w", err)
		return result
	}
	defer conn.Close()

	// Get connection state
	state := conn.ConnectionState()

	result.success = true
	result.version = state.Version
	result.cipherSuite = state.CipherSuite
	result.peerCertificates = len(state.PeerCertificates)

	return result
}

// testTLSHandshakeWithSNI performs a TLS handshake with custom SNI.
func testTLSHandshakeWithSNI(host string, port int, sni string, timeout time.Duration) *tlsTestResult {
	result := &tlsTestResult{
		serverName: sni,
	}

	address := fmt.Sprintf("%s:%d", host, port)

	// Create TLS configuration with custom SNI
	config := &tls.Config{
		ServerName: sni,
	}

	// Establish connection with timeout
	dialer := &net.Dialer{
		Timeout: timeout,
	}

	conn, err := tls.DialWithDialer(dialer, "tcp", address, config)
	if err != nil {
		result.err = fmt.Errorf("TLS handshake failed: %w", err)
		return result
	}
	defer conn.Close()

	// Get connection state
	state := conn.ConnectionState()

	result.success = true
	result.version = state.Version
	result.cipherSuite = state.CipherSuite
	result.peerCertificates = len(state.PeerCertificates)

	return result
}

// testPostgresSTARTTLS performs a STARTTLS handshake with a PostgreSQL server.
func testPostgresSTARTTLS(host string, port int, timeout time.Duration) *tlsTestResult {
	result := &tlsTestResult{
		serverName: host,
	}

	address := fmt.Sprintf("%s:%d", host, port)

	// Establish plain TCP connection
	dialer := &net.Dialer{
		Timeout: timeout,
	}

	conn, err := dialer.Dial("tcp", address)
	if err != nil {
		result.err = fmt.Errorf("failed to connect to postgres: %w", err)
		return result
	}
	defer conn.Close()

	// Send PostgreSQL STARTTLS request (SSLRequest message)
	// Message format: length (4 bytes) + SSL request code (4 bytes)
	sslRequest := []byte{0x00, 0x00, 0x00, 0x08, 0x04, 0xd2, 0x16, 0x2f}

	_, err = conn.Write(sslRequest)
	if err != nil {
		result.err = fmt.Errorf("failed to send SSL request: %w", err)
		return result
	}

	// Read response (should be 'S' for SSL supported)
	response := make([]byte, 1)
	_, err = conn.Read(response)
	if err != nil {
		result.err = fmt.Errorf("failed to read SSL response: %w", err)
		return result
	}

	if response[0] != 'S' {
		result.err = fmt.Errorf("server does not support SSL (response: %c)", response[0])
		return result
	}

	// Upgrade to TLS
	tlsConfig := &tls.Config{
		ServerName: host,
	}

	tlsConn := tls.Client(conn, tlsConfig)
	err = tlsConn.Handshake()
	if err != nil {
		result.err = fmt.Errorf("TLS handshake failed: %w", err)
		return result
	}

	// Get connection state
	state := tlsConn.ConnectionState()

	result.success = true
	result.version = state.Version
	result.cipherSuite = state.CipherSuite
	result.peerCertificates = len(state.PeerCertificates)

	return result
}

// testLDAPSTARTTLS performs a STARTTLS handshake with an LDAP server.
func testLDAPSTARTTLS(host string, port int, timeout time.Duration) *tlsTestResult {
	result := &tlsTestResult{
		serverName: host,
	}

	address := fmt.Sprintf("%s:%d", host, port)

	// Establish plain TCP connection
	dialer := &net.Dialer{
		Timeout: timeout,
	}

	conn, err := dialer.Dial("tcp", address)
	if err != nil {
		result.err = fmt.Errorf("failed to connect to LDAP: %w", err)
		return result
	}
	defer conn.Close()

	// Send LDAP STARTTLS Extended Operation request
	// This is a simplified LDAP STARTTLS request (BER encoded)
	startTLSRequest := []byte{
		0x30, 0x1d, // SEQUENCE, length 29
		0x02, 0x01, 0x01, // messageID: 1
		0x77, 0x18, // extendedReq, length 24
		0x80, 0x16, // requestName, length 22
		0x31, 0x2e, 0x33, 0x2e, 0x36, 0x2e, 0x31, 0x2e, // "1.3.6.1.4.1.1466.20037"
		0x34, 0x2e, 0x31, 0x2e, 0x31, 0x34, 0x36, 0x36,
		0x2e, 0x32, 0x30, 0x30, 0x33, 0x37,
	}

	_, err = conn.Write(startTLSRequest)
	if err != nil {
		result.err = fmt.Errorf("failed to send STARTTLS request: %w", err)
		return result
	}

	// Read response
	reader := bufio.NewReader(conn)
	response := make([]byte, 1024)
	n, err := reader.Read(response)
	if err != nil {
		result.err = fmt.Errorf("failed to read STARTTLS response: %w", err)
		return result
	}

	// Check if response indicates success (simplified check)
	if n < 10 {
		result.err = fmt.Errorf("invalid STARTTLS response length: %d", n)
		return result
	}

	// Upgrade to TLS
	tlsConfig := &tls.Config{
		ServerName: host,
	}

	tlsConn := tls.Client(conn, tlsConfig)
	err = tlsConn.Handshake()
	if err != nil {
		result.err = fmt.Errorf("TLS handshake failed: %w", err)
		return result
	}

	// Get connection state
	state := tlsConn.ConnectionState()

	result.success = true
	result.version = state.Version
	result.cipherSuite = state.CipherSuite
	result.peerCertificates = len(state.PeerCertificates)

	return result
}
