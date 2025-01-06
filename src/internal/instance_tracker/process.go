package instance_tracker

import (
	"os/exec"
	"time"
)

type RcloneProcess struct {
	PID         int
	Command     *exec.Cmd
	BackendName string
	StartedAt   time.Time
	GracePeriod time.Duration
	Environment map[string]string
}
