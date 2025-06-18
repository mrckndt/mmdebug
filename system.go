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
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/prometheus/procfs"
	"golang.org/x/sys/unix"
)

// TODO: Review if timeouts/context are needed for system operations
// Current 30-second timeouts might be overkill for local filesystem reads.
// Consider simplifying by removing context/timeout handling if operations
// are fast enough and don't benefit from cancellation.


// Core data structures
type SysctlConfig struct {
	Name     string
	Expected string
	Type     string
}

type UlimitConfig struct {
	Resource     int
	Name         string
	ExpectedSoft uint64
	ExpectedHard uint64
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
	expectedSoft uint64
	expectedHard uint64
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
		{unix.RLIMIT_NOFILE, "nofile", 65536, 65536},
		{unix.RLIMIT_NPROC, "nproc", 8192, 8192},
	}
}

// readSysctl reads a sysctl value with timeout
func readSysctl(ctx context.Context, name string) (string, error) {
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
func GetSysctls() ([]sysctlInfo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("sysctl only supported on Linux")
	}

	configs := defaultSysctlConfigs()
	results := make([]sysctlInfo, 0, len(configs))


	for _, config := range configs {
		actual, err := readSysctl(ctx, config.Name)
		if err != nil {
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
func GetUlimits() ([]ulimitInfo, error) {
	if runtime.GOOS != "linux" {
		return nil, fmt.Errorf("ulimits only supported on Linux")
	}

	configs := defaultUlimitConfigs()
	results := make([]ulimitInfo, 0, len(configs))


	for _, config := range configs {
		var limit unix.Rlimit
		if err := unix.Getrlimit(config.Resource, &limit); err != nil {
			continue
		}

		results = append(results, ulimitInfo{
			resourceName: config.Name,
			softLimit:    limit.Cur,
			hardLimit:    limit.Max,
			expectedSoft: config.ExpectedSoft,
			expectedHard: config.ExpectedHard,
		})
	}

	return results, nil
}

// GetMattermostProcessEnv gets Mattermost process environment variables
func GetMattermostProcessEnv() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	proc, err := findMattermostProcess(ctx)
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
func findMattermostProcess(ctx context.Context) (*procfs.Proc, error) {
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
func PrintSysctls() error {
	sysctls, err := GetSysctls()
	if err != nil {
		return err
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Parameter", "Expected", "Actual", "Status"})

	for _, sysctl := range sysctls {
		status := text.Colors{text.Bold, text.FgRed}.Sprint("FAIL")
		actual := text.Colors{text.Bold, text.FgRed}.Sprint(sysctl.actual)
		if sysctl.matches {
			status = text.Colors{text.Bold, text.FgGreen}.Sprint("OK")
			actual = text.Colors{text.Bold, text.FgGreen}.Sprint(sysctl.actual)
		}
		t.AppendRow(table.Row{
			sysctl.name,
			sysctl.expected,
			actual,
			status,
		})
	}

	t.SetStyle(table.StyleDefault)
	fmt.Printf("%s\n", text.Colors{text.Bold}.Sprint("Sysctl Parameters:"))
	t.Render()

	return nil
}

func PrintUlimits() error {
	ulimits, err := GetUlimits()
	if err != nil {
		return err
	}

	t := table.NewWriter()
	t.SetOutputMirror(os.Stdout)
	t.AppendHeader(table.Row{"Resource", "Type", "Expected", "Actual", "Status"})

	for _, limit := range ulimits {
		// Soft limit row
		softActual := formatUlimitValue(limit.softLimit)
		softMatches := limit.softLimit >= limit.expectedSoft || limit.softLimit == unix.RLIM_INFINITY
		softStatus := text.Colors{text.Bold, text.FgRed}.Sprint("FAIL")
		softActualColored := text.Colors{text.Bold, text.FgRed}.Sprint(softActual)
		if softMatches {
			softStatus = text.Colors{text.Bold, text.FgGreen}.Sprint("OK")
			softActualColored = text.Colors{text.Bold, text.FgGreen}.Sprint(softActual)
		}

		t.AppendRow(table.Row{
			limit.resourceName,
			"soft",
			limit.expectedSoft,
			softActualColored,
			softStatus,
		})

		// Hard limit row
		hardActual := formatUlimitValue(limit.hardLimit)
		hardMatches := limit.hardLimit >= limit.expectedHard || limit.hardLimit == unix.RLIM_INFINITY
		hardStatus := text.Colors{text.Bold, text.FgRed}.Sprint("FAIL")
		hardActualColored := text.Colors{text.Bold, text.FgRed}.Sprint(hardActual)
		if hardMatches {
			hardStatus = text.Colors{text.Bold, text.FgGreen}.Sprint("OK")
			hardActualColored = text.Colors{text.Bold, text.FgGreen}.Sprint(hardActual)
		}

		t.AppendRow(table.Row{
			limit.resourceName,
			"hard",
			limit.expectedHard,
			hardActualColored,
			hardStatus,
		})
	}

	t.SetStyle(table.StyleDefault)
	fmt.Printf("%s\n", text.Colors{text.Bold}.Sprint("Resource Limits:"))
	t.Render()

	return nil
}

func PrintMattermostProcessEnv() error {
	envVars, err := GetMattermostProcessEnv()
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
	fmt.Printf("%s\n", text.Colors{text.Bold}.Sprintf("Mattermost Process Environment Variables (%d total):", len(envVars)))
	t.Render()

	return nil
}

