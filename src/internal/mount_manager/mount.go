package mount_manager

import (
	"github.com/rs/zerolog"
	"rclone-manager/internal/config"
	"rclone-manager/internal/constants"
	"rclone-manager/internal/instance_tracker"
	"sync"
	"time"
)

type MountProcess struct {
	instance_tracker.RcloneProcess
	MountPoint string
}

var tracker instance_tracker.InstanceTracker[MountProcess]

func InitializeMountEndpoints(conf *config.Config, logger zerolog.Logger, processLock *sync.Mutex) {
	processLock.Lock()
	defer processLock.Unlock()

	if len(conf.Mounts) == 0 {
		logger.Debug().Msg("No rclone mounts endpoints defined... Skipping starting any")
		return
	}
	Cleanup(conf, logger)
	logger.Info().Msg("Initializing all mounts endpoints")
	setupMountsFromConfig(conf, logger)

	go MonitorMountProcesses(logger)
}

func StartMountWithRetries(instance *MountProcess, logger zerolog.Logger) *MountProcess {
	retries := 0
	for retries < 3 {
		ensureExists(instance.MountPoint, logger)
		cmd := createMountCommand(instance)
		instance.Command = cmd
		err := cmd.Start()
		if err == nil {
			logger.Info().
				Str(constants.LogBackend, instance.BackendName).
				Str(constants.LogMountPoint, instance.MountPoint).
				Msg("Mount started successfully.")
			instance.PID = cmd.Process.Pid
			instance.StartedAt = time.Now()
			instance.GracePeriod = 10 * time.Second
			tracker.Track(instance.BackendName, instance)
			go func() {
				err := cmd.Wait()
				if err != nil {
					logger.Warn().AnErr(constants.LogError, err).
						Str(constants.LogBackend, instance.BackendName).
						Msg("Mount process exited with error.")
				} else {
					logger.Info().
						Str(constants.LogBackend, instance.BackendName).
						Msg("Mount process exited normally.")
				}
			}()
			return instance
		}
		logger.Warn().AnErr(constants.LogError, err).Msgf("Mount failed. Retrying %d/3...", retries+1)
		retries++
		time.Sleep(5 * time.Second)
	}
	logger.Error().Str(constants.LogBackend, instance.BackendName).Msg("Failed to start Mount after 3 attempts.")
	return nil
}

func StopMount(instance *MountProcess, logger zerolog.Logger) {
	logger.Info().Str(constants.LogBackend, instance.BackendName).Msg("Stopping mount process...")
	UnmountEndpoint(instance, logger)
	if err := instance.Command.Process.Kill(); err == nil {
		tracker.Untrack(instance.BackendName)
		logger.Info().Int(constants.LogPid, instance.PID).Str(constants.LogBackend, instance.BackendName).Msg("Mount process stopped")
	} else {
		logger.Warn().AnErr(constants.LogError, err).Int(constants.LogPid, instance.PID).Str(constants.LogBackend, instance.BackendName).Msg("Failed to stop mount process")
	}
}

func Cleanup(config *config.Config, logger zerolog.Logger) {
	shouldMonitorProcesses = false
	logger.Info().Msg("Cleaning up all rclone mount processes")
	tracker.Range(func(key, value interface{}) bool {
		instance := value.(*MountProcess)
		StopMount(instance, logger)
		return true
	})
	UnmountAllByPath(config, logger)
}

func ReconcileMounts(conf *config.Config, logger zerolog.Logger, processLock *sync.Mutex) {
	processLock.Lock()
	defer processLock.Unlock()

	logger.Info().Msg("Reconciling mounts...")

	removeStaleMounts(conf, logger)
	setupMountsFromConfig(conf, logger)
}

func UnmountEndpoint(mount *MountProcess, logger zerolog.Logger) {
	cmd := createFuseUnmountCommand(&MountProcess{MountPoint: mount.MountPoint})
	err := cmd.Run()
	if err != nil {
		logger.Debug().AnErr(constants.LogError, err).
			Str(constants.LogMountPoint, mount.MountPoint).
			Msg("Failed to unmount path. It may not be mounted.")
	} else {
		logger.Info().Str(constants.LogMountPoint, mount.MountPoint).
			Msg("Unmounted successfully.")
	}
}

func UnmountAllByPath(conf *config.Config, logger zerolog.Logger) {
	logger.Info().Msg("Unmounting all paths listed in config...")

	for _, mount := range conf.Mounts {
		logger.Info().Str(constants.MountPoint, mount.MountPoint).Msg("Unmounting...")
		UnmountEndpoint(&MountProcess{MountPoint: mount.MountPoint}, logger)
	}
}
