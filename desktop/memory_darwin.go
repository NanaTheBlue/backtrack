//go:build darwin

package main

import (
	"syscall"
)

// getProcessRSS returns the resident set size (RAM reported by Activity Monitor) in bytes
func getProcessRSS() (uint64, error) {
	var r syscall.Rusage
	if err := syscall.Getrusage(syscall.RUSAGE_SELF, &r); err != nil {
		return 0, err
	}
	// On macOS (Darwin), ru_maxrss is reported in bytes
	return uint64(r.Maxrss), nil
}

