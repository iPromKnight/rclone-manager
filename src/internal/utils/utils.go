package utils

import (
	"fmt"
	"os"
	"syscall"
)

func ProcessIsRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	// Check for process existence
	if err := process.Signal(syscall.Signal(0)); err == nil {
		return true
	}

	// Additional check for /proc to avoid race condition
	_, err = os.Stat(fmt.Sprintf("/proc/%d", pid))
	return err == nil
}
