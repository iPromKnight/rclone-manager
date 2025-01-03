package rclone_manager

import (
	"github.com/rs/zerolog"
	"net/http"
	"os"
	"os/exec"
	"rclone-manager/internal/config"
	"rclone-manager/internal/mount_manager"
	"rclone-manager/internal/serve_manager"
	"rclone-manager/internal/utils"
	"sync"
	"syscall"
	"time"
)

type RCloneProcess struct {
	PID         int
	Command     *exec.Cmd
	StartedAt   time.Time
	GracePeriod time.Duration
}

var (
	processMap sync.Map
)

func InitializeRCD(logger zerolog.Logger) *config.Config {
	conf, err := config.LoadConfig()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to load configuration")
	}

	if len(conf.Serves) == 0 && len(conf.Mounts) == 0 {
		logger.Warn().Msg("No serves or mounts found in configuration. Running in RCD-only mode.")
	}

	mount_manager.UnmountAllByPath(conf, logger)

	logger.Info().Msg("Initializing Rclone in RCD mode")
	go StartRcloneRemoteDaemon(logger)
	go MonitorRCDProcess(conf, logger)

	WaitForRCD(logger, 10)

	if len(conf.Serves) > 0 {
		go serve_manager.InitializeServeEndpoints(conf, logger)
	}

	if len(conf.Mounts) > 0 {
		go mount_manager.InitializeMounts(conf, logger)
	}

	return conf
}

func StartRcloneRemoteDaemon(logger zerolog.Logger) *RCloneProcess {
	cmd := exec.Command("rclone", "rcd")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	err := cmd.Start()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to start rclone RCD")
		return nil
	}

	go func() {
		err := cmd.Wait()
		if err != nil {
			logger.Warn().Err(err).Msg("Rclone RCD exited unexpectedly")
		} else {
			logger.Warn().Msg("Rclone RCD exited gracefully")
		}
	}()

	rcloneProcess := &RCloneProcess{
		PID:         cmd.Process.Pid,
		Command:     cmd,
		StartedAt:   time.Now(),
		GracePeriod: 10 * time.Second,
	}

	processMap.Store("RCD", rcloneProcess)
	logger.Info().Int("pid", cmd.Process.Pid).Msg("Started rclone RCD process")
	return rcloneProcess
}

func StopRcloneRemoteDaemon(conf *config.Config, logger zerolog.Logger) {
	shouldMonitorProcesses = false
	if rcd, ok := processMap.Load("RCD"); ok {
		if !utils.ProcessIsRunning(rcd.(*RCloneProcess).PID) {
			logger.Warn().Msg("RCD process is not running")
			mount_manager.UnmountAllByPath(conf, logger)
			return
		}

		serve_manager.Cleanup(logger)
		mount_manager.StopAllMountsViaRCD(logger)

		err := rcd.(*RCloneProcess).Command.Process.Signal(syscall.SIGINT)
		if err != nil {
			return
		}
		processMap.Delete("RCD")
		logger.Info().Msg("Stopped rclone RCD process")
	}
}

func PingRCD(logger zerolog.Logger) bool {
	resp, err := http.Get("http://localhost:5572")
	if err != nil || resp.StatusCode != http.StatusOK {
		return false
	}
	logger.Debug().Msg("Rclone RCD is responsive")
	return true
}

func WaitForRCD(logger zerolog.Logger, maxRetries int) {
	for i := 0; i < maxRetries; i++ {
		if PingRCD(logger) {
			logger.Info().Msg("Rclone RCD is ready for mounts")
			return
		}
		logger.Warn().Msgf("Rclone RCD not ready. Retrying... (%d/%d)", i+1, maxRetries)
		time.Sleep(5 * time.Second)
	}

	logger.Fatal().Msg("Rclone RCD failed to start after retries. Exiting...")
}
