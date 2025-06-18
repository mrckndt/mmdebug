# CLAUDE.md

## Project: mmdebug

A Go-based (network) debugging and system diagnostics tool for Mattermost environments.

## Build & Run Commands

```bash
# Build the project
go build -o mmdebug .

# Build for Linux (cross-compile)
GOOS=linux GOARCH=amd64 go build -o mmdebug-linux-amd64 .

# Run with various modes
./mmdebug -host example.com -port 443 -mode tcp
./mmdebug -host example.com -port 443 -mode tls
./mmdebug -mode ulimits
./mmdebug -mode mm-env
./mmdebug -mode sysctl
```

## Project Structure
- `net.go` - Network connectivity testing functions
- `tls.go` - TLS/SSL connection testing
- `system.go` - System diagnostics (ulimits, environment, sysctl)
- `system_other.go` - Stubs of system.go functions for non-Linux platforms

## Dependencies & code otherwise in use
- OpenSSL for taking inspiration on TLS handshake tests, not a dependency on its own - https://github.com/openssl/openssl
- profcs for everything related to `/proc` on Linux - https://github.com/prometheus/procfs
- go-pretty for drawing tables, coloring text/output, showing progress and listing file trees - https://github.com/jedib0t/go-pretty
