# mmdebug

A Go-based network debugging and system diagnostics tool for Mattermost environments.

## Features

- **Network Connectivity Testing**: TCP connection testing
- **TLS/SSL Analysis**: Comprehensive TLS handshake testing with detailed certificate information
- **System Diagnostics**: System limits, environment variables, and kernel parameters
- **Cross-Platform**: Supports Linux, macOS, and other Unix-like systems

## Installation

```bash
# Build from source
go build -o mmdebug .

# Cross-compile for Linux
GOOS=linux GOARCH=amd64 go build -o mmdebug-linux-amd64 .
```

## Usage

### Network Testing

```bash
# Test TCP connection
./mmdebug -host example.com -port 443 -mode tcp
```

### TLS Testing

```bash
# Basic TLS handshake test
./mmdebug -host example.com -port 443 -mode tls

# TLS test ignoring certificate validation
./mmdebug -host example.com -port 443 -mode tls-insecure

# TLS test with custom SNI
./mmdebug -host example.com -port 443 -mode tls-sni -sni custom.example.com

# PostgreSQL STARTTLS test
./mmdebug -host postgres.example.com -port 5432 -mode tls-postgres

# LDAP STARTTLS test
./mmdebug -host ldap.example.com -port 389 -mode tls-ldap
```

### System Diagnostics

```bash
# Check system resource limits
./mmdebug -mode ulimits

# Display Mattermost environment variables
./mmdebug -mode mm-env

# Show kernel parameters
./mmdebug -mode sysctl
```

## Command Line Options

- `-host`: Target hostname or IP address (required for network tests)
- `-port`: Target port number (default: 443)
- `-timeout`: Connection timeout duration (default: 10s)
- `-mode`: Test mode (see modes below)
- `-sni`: Custom SNI for TLS connections (required for tls-sni mode)

## Test Modes

| Mode | Description |
|------|-------------|
| `tcp` | TCP connection test |
| `tls` | TLS handshake with certificate validation |
| `tls-insecure` | TLS handshake without certificate validation |
| `tls-sni` | TLS handshake with custom SNI |
| `tls-postgres` | PostgreSQL STARTTLS test |
| `tls-ldap` | LDAP STARTTLS test |
| `ulimits` | System resource limits |
| `mm-env` | Mattermost environment variables |
| `sysctl` | Kernel parameters |

## TLS Output

TLS tests provide detailed information including:
- TLS version (1.0, 1.1, 1.2, 1.3)
- Cipher suite
- Server name
- Number of peer certificates

## Dependencies

- [procfs](https://github.com/prometheus/procfs) - Linux `/proc` filesystem handling
- [go-pretty](https://github.com/jedib0t/go-pretty) - Table formatting and colored output

## License

This project is part of the Mattermost ecosystem. See [Mattermost documentation](https://docs.mattermost.com/) for more information.