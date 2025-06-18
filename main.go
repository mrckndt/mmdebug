package main

import (
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

func main() {
	var (
		host    = flag.String("host", "", "Host to connect to")
		port    = flag.Int("port", 443, "Port to connect to")
		timeout = flag.Duration("timeout", 10*time.Second, "Connection timeout")
		mode    = flag.String("mode", "tcp", "Test mode: tcp, tls, tls-insecure, tls-sni, tls-postgres, tls-ldap, ulimits, mm-env, sysctl")
		sni     = flag.String("sni", "", "Custom SNI for TLS connections")
	)

	flag.Parse()

	if *host == "" && *mode != "ulimits" && *mode != "mm-env" && *mode != "sysctl" {
		fmt.Fprintf(os.Stderr, "Error: host is required\n")
		flag.Usage()
		os.Exit(1)
	}

	switch strings.ToLower(*mode) {
	case "tcp":
		err := testTCPConnection(*host, *port, *timeout)
		printTCPResult(*host, *port, err)
		if err != nil {
			os.Exit(1)
		}


	case "tls":
		result := testTLSHandshake(*host, *port, *timeout)
		printTLSResult(result, *host, *port)
		if !result.success {
			os.Exit(1)
		}

	case "tls-insecure":
		result := testTLSHandshakeInsecure(*host, *port, *timeout)
		printTLSResult(result, *host, *port)
		if !result.success {
			os.Exit(1)
		}

	case "tls-sni":
		if *sni == "" {
			fmt.Fprintf(os.Stderr, "Error: SNI is required for tls-sni mode\n")
			os.Exit(1)
		}
		result := testTLSHandshakeWithSNI(*host, *port, *sni, *timeout)
		printTLSResult(result, *host, *port)
		if !result.success {
			os.Exit(1)
		}

	case "tls-postgres":
		result := testPostgresSTARTTLS(*host, *port, *timeout)
		printTLSResult(result, *host, *port)
		if !result.success {
			os.Exit(1)
		}

	case "tls-ldap":
		result := testLDAPSTARTTLS(*host, *port, *timeout)
		printTLSResult(result, *host, *port)
		if !result.success {
			os.Exit(1)
		}

	case "ulimits":
		err := PrintUlimits()
		if err != nil {
			fmt.Printf("Failed to get ulimits: %v\n", err)
			os.Exit(1)
		}

	case "mm-env":
		err := PrintMattermostEnvironmentVariables()
		if err != nil {
			fmt.Printf("Failed to get Mattermost environment variables: %v\n", err)
			os.Exit(1)
		}

	case "sysctl":
		err := PrintSysctls()
		if err != nil {
			fmt.Printf("Failed to get sysctl parameters: %v\n", err)
			os.Exit(1)
		}

	default:
		fmt.Fprintf(os.Stderr, "Error: unknown mode '%s'\n", *mode)
		fmt.Fprintf(os.Stderr, "Available modes: tcp, tls, tls-insecure, tls-sni, tls-postgres, tls-ldap, ulimits, mm-env, sysctl\n")
		os.Exit(1)
	}
}

// printTLSResult outputs TLS test results in a formatted way.
func printTLSResult(result *tlsTestResult, host string, port int) {
	if result.success {
		fmt.Printf("TLS connection to %s:%d successful\n", host, port)
		fmt.Printf("  TLS Version: %s\n", tlsVersionString(result.version))
		fmt.Printf("  Cipher Suite: %s\n", cipherSuiteString(result.cipherSuite))
		fmt.Printf("  Server Name: %s\n", result.serverName)
		fmt.Printf("  Peer Certificates: %d\n", result.peerCertificates)
	} else {
		fmt.Printf("TLS connection to %s:%d failed: %v\n", host, port, result.err)
	}
}

// tlsVersionString converts a TLS version number to a human-readable string.
func tlsVersionString(version uint16) string {
	switch version {
	case 0x0300:
		return "SSL 3.0"
	case 0x0301:
		return "TLS 1.0"
	case 0x0302:
		return "TLS 1.1"
	case 0x0303:
		return "TLS 1.2"
	case 0x0304:
		return "TLS 1.3"
	default:
		return fmt.Sprintf("Unknown (0x%04x)", version)
	}
}

// cipherSuiteString converts a cipher suite number to a human-readable string.
func cipherSuiteString(suite uint16) string {
	suites := map[uint16]string{
		0x0004: "TLS_RSA_WITH_RC4_128_MD5",
		0x0005: "TLS_RSA_WITH_RC4_128_SHA",
		0x000a: "TLS_RSA_WITH_3DES_EDE_CBC_SHA",
		0x002f: "TLS_RSA_WITH_AES_128_CBC_SHA",
		0x0035: "TLS_RSA_WITH_AES_256_CBC_SHA",
		0x003c: "TLS_RSA_WITH_AES_128_CBC_SHA256",
		0x003d: "TLS_RSA_WITH_AES_256_CBC_SHA256",
		0x009c: "TLS_RSA_WITH_AES_128_GCM_SHA256",
		0x009d: "TLS_RSA_WITH_AES_256_GCM_SHA384",
		0xc007: "TLS_ECDHE_ECDSA_WITH_RC4_128_SHA",
		0xc009: "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA",
		0xc00a: "TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA",
		0xc011: "TLS_ECDHE_RSA_WITH_RC4_128_SHA",
		0xc013: "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA",
		0xc014: "TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA",
		0xc023: "TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA256",
		0xc024: "TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA384",
		0xc027: "TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA256",
		0xc028: "TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA384",
		0xc02b: "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
		0xc02c: "TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
		0xc02f: "TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
		0xc030: "TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
		0x1301: "TLS_AES_128_GCM_SHA256",
		0x1302: "TLS_AES_256_GCM_SHA384",
		0x1303: "TLS_CHACHA20_POLY1305_SHA256",
	}

	if name, ok := suites[suite]; ok {
		return name
	}
	return fmt.Sprintf("Unknown (0x%04x)", suite)
}
