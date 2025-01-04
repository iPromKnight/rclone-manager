package rclone_manager

import (
	"github.com/rs/zerolog"
	"os/exec"
	"rclone-manager/internal/config"
	"rclone-manager/internal/constants"
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
	processMap  sync.Map
	processLock sync.Mutex
)

var (
	LoadedConfig *config.Config
)

func InitializeRCD(logger zerolog.Logger) {

	conf, err := config.LoadConfig()
	if err != nil {

		logger.Fatal().Err(err).Msg("Failed to load configuration")
	}

	if len(conf.Serves) == 0 && len(conf.Mounts) == 0 {
		logger.Warn().Msg("No serves or mounts found in configuration. Running in RCD-only mode.")
	}

	processLock.Lock()
	defer processLock.Unlock()

	LoadedConfig = conf

	mount_manager.UnmountAllByPath(conf, logger)

	logger.Info().Msg("Initializing Rclone in RCD mode")
	go StartRcloneRemoteDaemon(logger)
	go MonitorRCDProcess(conf, logger)

	waitForRCD(logger, 10)

	propagateRCDEnv(logger)

	if len(conf.Serves) > 0 {
		go serve_manager.InitializeServeEndpoints(conf, logger, &processLock)
	}

	if len(conf.Mounts) > 0 {
		go mount_manager.InitializeMounts(conf, logger, &processLock)
	}

	startFileWatcher(logger)
}

func StartRcloneRemoteDaemon(logger zerolog.Logger) *RCloneProcess {
	cmd := createStartRcdCommand()
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

	trackRCD(rcloneProcess)
	logger.Debug().Int(constants.LogPid, cmd.Process.Pid).Msg("Started rclone RCD process")
	return rcloneProcess
}

func StopRcloneRemoteDaemon(logger zerolog.Logger) {
	shouldMonitorProcesses = false
	if rcd, ok := processMap.Load(constants.Rcd); ok {
		if !utils.ProcessIsRunning(rcd.(*RCloneProcess).PID) {
			logger.Warn().Msg("RCD process is not running")
			mount_manager.UnmountAllByPath(LoadedConfig, logger)
			return
		}

		serve_manager.Cleanup(logger)
		mount_manager.StopAllMountsViaRCD(logger)

		err := rcd.(*RCloneProcess).Command.Process.Signal(syscall.SIGINT)
		if err != nil {
			return
		}
		untrackRCD()
		logger.Info().Msg("Stopped rclone RCD process")
	}
}
