//go:build !linux

package main

import (
	"fmt"
	"runtime"
)

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

// Stub implementations for non-Linux systems
func GetSysctls() ([]sysctlInfo, error) {
	return nil, fmt.Errorf("sysctl reading is only supported on Linux, current OS: %s", runtime.GOOS)
}

func GetUlimits() ([]ulimitInfo, error) {
	return nil, fmt.Errorf("ulimits are only supported on Linux, current OS: %s", runtime.GOOS)
}

func GetMattermostEnv() ([]string, error) {
	return nil, fmt.Errorf("reading Mattermost process environment variables is only supported on Linux, current OS: %s", runtime.GOOS)
}

func PrintSysctls() error {
	return fmt.Errorf("sysctl reading is only supported on Linux, current OS: %s", runtime.GOOS)
}

func PrintUlimits() error {
	return fmt.Errorf("ulimits are only supported on Linux, current OS: %s", runtime.GOOS)
}

func PrintMattermostEnv() error {
	return fmt.Errorf("reading Mattermost process environment variables is only supported on Linux, current OS: %s", runtime.GOOS)
}

