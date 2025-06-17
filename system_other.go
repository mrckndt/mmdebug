//go:build !linux

package main

import (
	"context"
	"fmt"
	"runtime"
	"time"
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

// SystemChecker - stub for non-Linux systems
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
	// No-op for non-Linux
}

// GetSysctls returns an error on non-Linux systems
func (s *SystemChecker) GetSysctls(ctx context.Context) ([]sysctlInfo, error) {
	return nil, fmt.Errorf("sysctl reading is only supported on Linux, current OS: %s", runtime.GOOS)
}

// GetUlimits returns an error on non-Linux systems
func (s *SystemChecker) GetUlimits() ([]ulimitInfo, error) {
	return nil, fmt.Errorf("ulimits are only supported on Linux, current OS: %s", runtime.GOOS)
}

// GetMattermostEnv returns an error on non-Linux systems
func (s *SystemChecker) GetMattermostEnv(ctx context.Context) ([]string, error) {
	return nil, fmt.Errorf("reading Mattermost process environment variables is only supported on Linux, current OS: %s", runtime.GOOS)
}

// PrintSysctls returns an error on non-Linux systems
func (s *SystemChecker) PrintSysctls(ctx context.Context) error {
	return fmt.Errorf("sysctl reading is only supported on Linux, current OS: %s", runtime.GOOS)
}

// PrintUlimits returns an error on non-Linux systems
func (s *SystemChecker) PrintUlimits() error {
	return fmt.Errorf("ulimits are only supported on Linux, current OS: %s", runtime.GOOS)
}

// PrintMattermostEnv returns an error on non-Linux systems
func (s *SystemChecker) PrintMattermostEnv(ctx context.Context) error {
	return fmt.Errorf("reading Mattermost process environment variables is only supported on Linux, current OS: %s", runtime.GOOS)
}

// Legacy functions for backward compatibility
func GetMattermostEnvironmentVariables() ([]string, error) {
	return nil, fmt.Errorf("reading Mattermost process environment variables is only supported on Linux, current OS: %s", runtime.GOOS)
}

func GetSysctls() ([]sysctlInfo, error) {
	return nil, fmt.Errorf("sysctl reading is only supported on Linux, current OS: %s", runtime.GOOS)
}

func GetSysctlsWithConfig(configs []SysctlConfig) ([]sysctlInfo, error) {
	return nil, fmt.Errorf("sysctl reading is only supported on Linux, current OS: %s", runtime.GOOS)
}

func GetUlimits() ([]ulimitInfo, error) {
	return nil, fmt.Errorf("ulimits are only supported on Linux, current OS: %s", runtime.GOOS)
}

func GetUlimitsWithConfig(configs []UlimitConfig) ([]ulimitInfo, error) {
	return nil, fmt.Errorf("ulimits are only supported on Linux, current OS: %s", runtime.GOOS)
}

func PrintSysctls() error {
	return fmt.Errorf("sysctl reading is only supported on Linux, current OS: %s", runtime.GOOS)
}

func PrintUlimits() error {
	return fmt.Errorf("ulimits are only supported on Linux, current OS: %s", runtime.GOOS)
}

func PrintMattermostEnvironmentVariables() error {
	return fmt.Errorf("reading Mattermost process environment variables is only supported on Linux, current OS: %s", runtime.GOOS)
}
