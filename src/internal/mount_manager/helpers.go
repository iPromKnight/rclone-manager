package mount_manager

import (
	"fmt"
	"github.com/rs/zerolog"
	"os"
	"os/exec"
	"rclone-manager/internal/config"
	"rclone-manager/internal/constants"
	"rclone-manager/internal/environment"
	"rclone-manager/internal/instance_tracker"
)

func createMountCommand(instance *MountProcess) *exec.Cmd {
	backendArg := fmt.Sprintf("%s:", instance.BackendName)

	cmd := exec.Command(
		constants.Rclone, constants.Mount, backendArg, instance.MountPoint)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Env = environment.PrepareEnvironment(instance.Environment)

	return cmd
}

func createFuseUnmountCommand(instance *MountProcess) *exec.Cmd {
	cmd := exec.Command(constants.Fusermount, constants.FuseUnmount, instance.MountPoint)
	cmd.Stdout = os.Stdout
	return cmd
}

func setupMountsFromConfig(conf *config.Config, logger zerolog.Logger) {
	for _, mount := range conf.Mounts {
		instance := &MountProcess{
			MountPoint: mount.MountPoint,
			RcloneProcess: instance_tracker.RcloneProcess{
				BackendName: mount.BackendName,
				Environment: mount.Environment,
			},
		}
		if existing, ok := tracker.Get(mount.BackendName); ok {
			if existing.MountPoint != mount.MountPoint {
				logger.Warn().
					Str(constants.LogBackend, mount.BackendName).
					Msg("Mount config changed, restarting...")
				StopMount(existing, logger)
				StartMountWithRetries(instance, logger)
			}
		} else {
			logger.Info().
				Str(constants.LogBackend, mount.BackendName).
				Msg("New mount detected, starting...")
			StartMountWithRetries(instance, logger)
		}
	}
}

func removeStaleMounts(conf *config.Config, logger zerolog.Logger) {
	var staleKeys []interface{}

	tracker.Range(func(key, value interface{}) bool {
		instance := value.(*MountProcess)
		if !config.IsMountInConfig(instance.BackendName, conf) {
			logger.Warn().
				Str(constants.LogBackend, instance.BackendName).
				Msg("mount removed from config, stopping...")
			staleKeys = append(staleKeys, key)
		}
		return true
	})

	for _, key := range staleKeys {
		if instance, ok := tracker.Get(key.(string)); ok {
			StopMount(instance, logger)
		}
	}
}

func ensureExists(mountPoint string, logger zerolog.Logger) {
	if _, err := os.Stat(mountPoint); os.IsNotExist(err) {
		logger.Info().Str(constants.LogMountPoint, mountPoint).Msg("Creating mount point...")
		err := os.MkdirAll(mountPoint, 0777)
		if err != nil {
			logger.Error().Err(err).Str(constants.LogMountPoint, mountPoint).
				Msg("Failed to create mount point")
		} else {
			logger.Info().Str(constants.LogMountPoint, mountPoint).
				Msg("Mount point created successfully.")
		}
	}
}
