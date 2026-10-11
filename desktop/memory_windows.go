//go:build windows

package main

import (
	"unsafe"

	"golang.org/x/sys/windows"
)

type processMemoryCounters struct {
	cb                         uint32
	pageFaultCount             uint32
	peakWorkingSetSize         uintptr
	workingSetSize             uintptr
	quotaPeakPagedPoolUsage    uintptr
	quotaPagedPoolUsage        uintptr
	quotaPeakNonPagedPoolUsage uintptr
	quotaNonPagedPoolUsage     uintptr
	pagefileUsage              uintptr
	peakPagefileUsage          uintptr
}

var (
	modpsapi                 = windows.NewLazySystemDLL("psapi.dll")
	procGetProcessMemoryInfo = modpsapi.NewProc("GetProcessMemoryInfo")
)

// getProcessRSS returns the working set size (RAM reported by Windows Task Manager) in bytes
func getProcessRSS() (uint64, error) {
	handle := windows.CurrentProcess()

	var pmc processMemoryCounters
	pmc.cb = uint32(unsafe.Sizeof(pmc))
	r1, _, callErr := procGetProcessMemoryInfo.Call(
		uintptr(handle),
		uintptr(unsafe.Pointer(&pmc)),
		uintptr(pmc.cb),
	)
	if r1 == 0 {
		return 0, callErr
	}

	return uint64(pmc.workingSetSize), nil
}

