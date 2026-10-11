//go:build !linux && !windows && !darwin

package main

import (
	goruntime "runtime"
)

func getProcessRSS() (uint64, error) {
	var m goruntime.MemStats
	goruntime.ReadMemStats(&m)
	return m.Sys, nil
}

