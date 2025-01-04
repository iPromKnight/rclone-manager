package mount_manager

import (
	"github.com/rs/zerolog"
	"rclone-manager/internal/config"
	"rclone-manager/internal/constants"
	"sync"
	"time"
)

var (
	instanceMap   sync.Map
	currentRCDEnv map[string]interface{}
)

type MountedEndpoint struct {
	BackendName string
	MountPoint  string
	EnvVars     map[string]string
}

func SetRCDEnv(env map[string]interface{}) {
	currentRCDEnv = env
}

func InitializeMounts(conf *config.Config, logger zerolog.Logger, processLock *sync.Mutex) {
	processLock.Lock()
	defer processLock.Unlock()

	if len(conf.Mounts) == 0 {
		logger.Debug().Msg("No rclone mount endpoints defined... Skipping...")
		return
	}

	logger.Info().Msg("Initializing all Mounts")
	for _, mount := range conf.Mounts {
		instance := &MountedEndpoint{
			BackendName: mount.BackendName,
			MountPoint:  mount.MountPoint,
			EnvVars:     mount.Environment,
		}
		StartMountWithRetries(instance, logger)
	}
}

func StartMountWithRetries(instance *MountedEndpoint, logger zerolog.Logger) {
	retries := 0
	for retries < 3 {
		ensureMountPointExists(instance.MountPoint, logger)
		cmd := createMountCommand(instance, logger)
		if cmd != nil {
			err := cmd.Run()
			if err == nil {
				logger.Info().Str(constants.LogBackend, instance.BackendName).
					Str(constants.LogMountPoint, instance.MountPoint).
					Msg("Mount successful.")
				trackEndpoint(instance)
				return
			}
		}
		logger.Warn().Msgf("Mount failed. Retrying %d/3...", retries+1)
		retries++
		time.Sleep(5 * time.Second)
	}
	logger.Error().Str(constants.LogBackend, instance.BackendName).Msg("Failed to mount after 3 attempts.")
}

func StopAllMountsViaRCD(logger zerolog.Logger) {
	logger.Info().Msg("Unmounting all rclone mounts")
	cmd := createUnmountAllCommand()
	err := cmd.Run()
	if err != nil {
		logger.Error().AnErr(constants.LogError, err).Msg("Failed to unmount all rclone mounts")
		return
	}

	instanceMap.Clear()
	logger.Info().Msg("Unmounted all rclone mounts successfully.")
}

func UnmountInstanceViaRcdWithFuseFallback(instance *MountedEndpoint, logger zerolog.Logger) bool {
	logger.Info().Str(constants.LogBackend, instance.BackendName).
		Str(constants.LogMountPoint, instance.MountPoint).
		Msg("Unmounting Endpoint")

	cmd := createUnmountCommand(instance)
	err := cmd.Run()
	if err != nil {
		logger.Error().AnErr(constants.LogError, err).
			Msg("Failed to unmount via rc. Falling back to fusermount.")
		cmd := createFuseUnmountCommand(instance)
		err := cmd.Run()
		if err != nil {
			logger.Error().Str(constants.LogBackend, instance.BackendName).
				Str(constants.LogMountPoint, instance.MountPoint).
				AnErr(constants.LogError, err).
				Msg("Failed to unmount rclone mount via fuse fallback")
			return false
		}
	}

	logger.Info().Str(constants.LogBackend, instance.BackendName).
		Str(constants.LogMountPoint, instance.MountPoint).
		Msg("Unmounted mount-point successfully.")
	return true
}

func ReconcileMounts(conf *config.Config, logger zerolog.Logger, processLock *sync.Mutex) {
	processLock.Lock()
	defer processLock.Unlock()

	logger.Info().Msg("Reconciling mounts...")

	setupMountsFromConfig(conf, logger)
	removeStaleMounts(conf, logger)
}

func UnmountAllByPath(conf *config.Config, logger zerolog.Logger) {
	logger.Info().Msg("Unmounting all paths listed in config...")

	for _, mount := range conf.Mounts {
		logger.Info().Str(constants.MountPoint, mount.MountPoint).Msg("Unmounting...")
		cmd := createFuseUnmountCommand(&MountedEndpoint{MountPoint: mount.MountPoint})
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
}
