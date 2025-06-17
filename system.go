//go:build linux

package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/prometheus/procfs"
	"golang.org/x/sys/unix"
)


// Core data structures
type SysctlConfig struct {
	Name     string
	Expected string
	Type     string
}

type UlimitConfig struct {
	Resource int
	Name     string
	Expected uint64
}

type sysctlInfo struct {
	name     string
	expected string
	actual   string
	matches  bool
}

type ulimitInfo struct {
	resourceName string
	softLimit    uint64
	hardLimit    uint64
	expected     uint64
}

// SystemChecker - simplified single class for all operations
type SystemChecker struct {
	timeout time.Duration
	verbose bool
}

// NewSystemChecker creates a checker with optional timeout
func NewSystemChecker(timeout time.Duration, verbose bool) *SystemChecker {
	if timeout == 0 {
		timeout = 10 * time.Second
	}
	return &SystemChecker{timeout: timeout, verbose: verbose}
}

// log writes to stderr if verbose mode is enabled
func (s *SystemChecker) log(format string, args ...interface{}) {
	if s.verbose {
		fmt.Fprintf(os.Stderr, format+"\n", args...)
	}
}

// Default configurations
func defaultSysctlConfigs() []SysctlConfig {
	return []SysctlConfig{
		{"net.ipv4.ip_local_port_range", "1025 65000", "range"},
		{"net.ipv4.tcp_fin_timeout", "30", "int"},
		{"net.ipv4.tcp_tw_reuse", "1", "int"},
		{"net.core.somaxconn", "4096", "int"},
		{"net.ipv4.tcp_max_syn_backlog", "8192", "int"},
		{"vm.min_free_kbytes", "167772", "int"},
		{"net.ipv4.tcp_slow_start_after_idle", "0", "int"},
		{"net.ipv4.tcp_congestion_control", "bbr", "string"},
		{"net.core.default_qdisc", "fq", "string"},
		{"net.ipv4.tcp_notsent_lowat", "16384", "int"},
		{"net.ipv4.tcp_rmem", "4096 156250 625000", "range"},
		{"net.ipv4.tcp_wmem", "4096 156250 625000", "range"},
		{"net.core.rmem_max", "312500", "int"},
		{"net.core.wmem_max", "312500", "int"},
		{"net.core.rmem_default", "312500", "int"},
		{"net.core.wmem_default", "312500", "int"},
		{"net.ipv4.tcp_mem", "1638400 1638400 1638400", "range"},
	}
}

func defaultUlimitConfigs() []UlimitConfig {
	return []UlimitConfig{
		{unix.RLIMIT_NOFILE, "nofile", 65536},
		{unix.RLIMIT_NPROC, "nproc", 8192},
	}
}

// readSysctl reads a sysctl value with timeout
func (s *SystemChecker) readSysctl(ctx context.Context, name string) (string, error) {
	if name == "" {
		return "", fmt.Errorf("empty parameter name")
	}

	done := make(chan struct {
		value string
		err   error
	}, 1)

	go func() {
		fs, err := procfs.NewDefaultFS()
		if err != nil {
			done <- struct {
				value string
				err   error
			}{"", fmt.Errorf("procfs access failed: %w", err)}
			return
		}

		path := strings.ReplaceAll(name, ".", "/")

		// Try strings first, then integers
		if values, err := fs.SysctlStrings(path); err == nil && len(values) > 0 {
			done <- struct {
				value string
				err   error
			}{strings.Join(values, " "), nil}
			return
		}

		if values, err := fs.SysctlInts(path); err == nil && len(values) > 0 {
			parts := make([]string, len(values))
			for i, v := range values {
				parts[i] = fmt.Sprintf("%d", v)
			}
			done <- struct {
				value string
				err   error
			}{strings.Join(parts, " "), nil}
			return
		}

		done <- struct {
			value string
			err   error
		}{"", fmt.Errorf("parameter not found")}
	}()

	select {
	case result := <-done:
		return result.value, result.err
	case <-ctx.Done():
		return "", fmt.Errorf("timeout reading %s", name)
	}
}

// compareSysctl compares sysctl values (handles multi-value integers)
func compareSysctl(expected, actual string) bool {
	expected = strings.TrimSpace(expected)
	actual = strings.TrimSpace(actual)

	if expected == actual {
		return true
	}

	// Try multi-value integer comparison
	expectedFields := strings.Fields(expected)
	actualFields := strings.Fields(actual)

	if len(expectedFields) != len(actualFields) {
		return false
	}

	for i := 0; i < len(expectedFields); i++ {
		exp, err1 := strconv.Atoi(expectedFields[i])
		act, err2 := strconv.Atoi(actualFields[i])
		
		if err1 != nil || err2 != nil {
			return false // Fall back to string comparison, which already failed
		}
		
		if act < exp {
			return false
		}
	}
	
	return true
}

// GetSysctls retrieves and validates sysctl parameters
func (s *SystemChecker) GetSysctls(ctx context.Context) ([]sysctlInfo, error) {
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("sysctl only supported on Linux")
	}

	configs := defaultSysctlConfigs()
	results := make([]sysctlInfo, 0, len(configs))

	s.log("Checking %d sysctl parameters", len(configs))

	for _, config := range configs {
		actual, err := s.readSysctl(ctx, config.Name)
		if err != nil {
			s.log("Failed to read %s: %v", config.Name, err)
			actual = "not found"
		}

		results = append(results, sysctlInfo{
			name:     config.Name,
			expected: config.Expected,
			actual:   actual,
			matches:  actual != "not found" && compareSysctl(config.Expected, actual),
		})
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].name < results[j].name
	})

	return results, nil
}

// GetUlimits retrieves and validates ulimit information
func (s *SystemChecker) GetUlimits() ([]ulimitInfo, error) {
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("ulimits only supported on Linux")
	}

	configs := defaultUlimitConfigs()
	results := make([]ulimitInfo, 0, len(configs))

	s.log("Checking %d ulimit resources", len(configs))

	for _, config := range configs {
		var limit syscall.Rlimit
		if err := syscall.Getrlimit(config.Resource, &limit); err != nil {
			s.log("Failed to get %s: %v", config.Name, err)
			continue
		}

		results = append(results, ulimitInfo{
			resourceName: config.Name,
			softLimit:    limit.Cur,
			hardLimit:    limit.Max,
			expected:     config.Expected,
		})
	}

	return results, nil
}

// GetMattermostEnv gets Mattermost environment variables
func (s *SystemChecker) GetMattermostEnv(ctx context.Context) ([]string, error) {
	proc, err := s.findMattermostProcess(ctx)
	if err != nil {
		return nil, err
	}

	environ, err := proc.Environ()
	if err != nil {
		return nil, fmt.Errorf("failed to read environment for PID %d: %w", proc.PID, err)
	}

	var filtered []string
	for _, env := range environ {
		if strings.HasPrefix(env, "MM_") {
			filtered = append(filtered, env)
		}
	}

	if len(filtered) == 0 {
		return nil, fmt.Errorf("no MM_ environment variables found")
	}

	sort.Strings(filtered)
	return filtered, nil
}

// findMattermostProcess finds the mattermost process
func (s *SystemChecker) findMattermostProcess(ctx context.Context) (*procfs.Proc, error) {
	done := make(chan struct {
		proc *procfs.Proc
		err  error
	}, 1)

	go func() {
		fs, err := procfs.NewFS("/proc")
		if err != nil {
			done <- struct {
				proc *procfs.Proc
				err  error
			}{nil, fmt.Errorf("procfs access failed: %w", err)}
			return
		}

		procs, err := fs.AllProcs()
		if err != nil {
			done <- struct {
				proc *procfs.Proc
				err  error
			}{nil, fmt.Errorf("failed to get process list: %w", err)}
			return
		}

		for _, proc := range procs {
			if comm, err := proc.Comm(); err == nil && comm == "mattermost" {
				done <- struct {
					proc *procfs.Proc
					err  error
				}{&proc, nil}
				return
			}
		}

		done <- struct {
			proc *procfs.Proc
			err  error
		}{nil, fmt.Errorf("mattermost process not found")}
	}()

	select {
	case result := <-done:
		return result.proc, result.err
	case <-ctx.Done():
		return nil, fmt.Errorf("timeout finding mattermost process")
	}
}


func formatUlimitValue(value uint64) string {
	if value == unix.RLIM_INFINITY {
		return "unlimited"
	}
	return fmt.Sprintf("%d", value)
}

// Print functions
func (s *SystemChecker) PrintSysctls(ctx context.Context) error {
	sysctls, err := s.GetSysctls(ctx)
	if err != nil {
		return err
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Parameter", "Expected", "Actual", "Status"})

	for _, sysctl := range sysctls {
		status := text.FgRed.Sprint("FAIL")
		actual := text.FgRed.Sprint(sysctl.actual)
		if sysctl.matches {
			status = text.FgGreen.Sprint("OK")
			actual = text.FgGreen.Sprint(sysctl.actual)
		}
		t.AppendRow(table.Row{
			sysctl.name,
			sysctl.expected,
			actual,
			status,
		})
	}

	t.SetStyle(table.StyleDefault)
	fmt.Println("Sysctl Parameters:")
	t.Render()

	return nil
}

func (s *SystemChecker) PrintUlimits() error {
	ulimits, err := s.GetUlimits()
	if err != nil {
		return err
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Resource", "Expected", "Actual", "Status"})

	for _, limit := range ulimits {
		actualValue := formatUlimitValue(limit.softLimit)
		matches := limit.softLimit >= limit.expected || limit.softLimit == unix.RLIM_INFINITY
		status := text.FgRed.Sprint("FAIL")
		actual := text.FgRed.Sprint(actualValue)
		if matches {
			status = text.FgGreen.Sprint("OK")
			actual = text.FgGreen.Sprint(actualValue)
		}

		t.AppendRow(table.Row{
			limit.resourceName,
			limit.expected,
			actual,
			status,
		})
	}

	t.SetStyle(table.StyleDefault)
	fmt.Println("Resource Limits:")
	t.Render()

	return nil
}

func (s *SystemChecker) PrintMattermostEnv(ctx context.Context) error {
	envVars, err := s.GetMattermostEnv(ctx)
	if err != nil {
		return err
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Variable", "Value"})

	for _, env := range envVars {
		parts := strings.SplitN(env, "=", 2)
		name := parts[0]
		value := ""
		if len(parts) > 1 {
			value = parts[1]
		}

		t.AppendRow(table.Row{name, value})
	}

	t.SetStyle(table.StyleDefault)
	fmt.Printf("Mattermost Environment Variables (%d total):\n", len(envVars))
	t.Render()

	return nil
}

// Legacy functions for backward compatibility
func GetMattermostEnvironmentVariables() ([]string, error) {
	checker := NewSystemChecker(30*time.Second, false)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return checker.GetMattermostEnv(ctx)
}

func GetSysctls() ([]sysctlInfo, error) {
	checker := NewSystemChecker(10*time.Second, false)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return checker.GetSysctls(ctx)
}

func GetSysctlsWithConfig(configs []SysctlConfig) ([]sysctlInfo, error) {
	return GetSysctls() // Simplified - ignores custom configs for now
}

func GetUlimits() ([]ulimitInfo, error) {
	checker := NewSystemChecker(10*time.Second, false)
	return checker.GetUlimits()
}

func GetUlimitsWithConfig(configs []UlimitConfig) ([]ulimitInfo, error) {
	return GetUlimits() // Simplified - ignores custom configs for now
}

func PrintSysctls() error {
	checker := NewSystemChecker(10*time.Second, false)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return checker.PrintSysctls(ctx)
}

func PrintUlimits() error {
	checker := NewSystemChecker(10*time.Second, false)
	return checker.PrintUlimits()
}

func PrintMattermostEnvironmentVariables() error {
	checker := NewSystemChecker(30*time.Second, false)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	return checker.PrintMattermostEnv(ctx)
}
