package mount_manager

import (
	"fmt"
	"github.com/rs/zerolog"
	"os"
	"os/exec"
	"rclone-manager/internal/config"
	"rclone-manager/internal/constants"
	"syscall"
)

func trackEndpoint(instance *MountedEndpoint) {
	instanceMap.Store(instance.BackendName, instance)
}

func untrackEndpoint(instance *MountedEndpoint) {
	instanceMap.Delete(instance.BackendName)
}

func getMountedEndpoint(key interface{}) (*MountedEndpoint, bool) {
	if val, ok := instanceMap.Load(key); ok {
		instance, valid := val.(*MountedEndpoint)
		return instance, valid
	}
	return nil, false
}

func createMountCommand(instance *MountedEndpoint) *exec.Cmd {
	fsArg := fmt.Sprintf("%s%s:", constants.Fs, instance.BackendName)
	mountPointArg := fmt.Sprintf("%s%s", constants.MountPoint, instance.MountPoint)

	cmd := exec.Command(constants.Rclone, constants.Rc, constants.Mount, fsArg, mountPointArg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd
}

func createUnmountCommand(instance *MountedEndpoint) *exec.Cmd {
	mountPointArg := fmt.Sprintf("%s%s", constants.MountPoint, instance.MountPoint)

	cmd := exec.Command(constants.Rclone, constants.Rc, constants.Unmount, mountPointArg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd
}

func createUnmountAllCommand() *exec.Cmd {
	cmd := exec.Command(constants.Rclone, constants.Rc, constants.UnmountAll)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd
}

func createFuseUnmountCommand(instance *MountedEndpoint) *exec.Cmd {
	cmd := exec.Command(constants.Fusermount, constants.FuseUnmount, instance.MountPoint)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd
}

func ensureMountPointExists(mountPoint string, logger zerolog.Logger) {
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

func setupMountsFromConfig(conf *config.Config, logger zerolog.Logger) {
	for _, mount := range conf.Mounts {
		instance := &MountedEndpoint{
			BackendName: mount.BackendName,
			MountPoint:  mount.MountPoint,
		}
		if existing, ok := getMountedEndpoint(mount.BackendName); ok {
			if existing.MountPoint != instance.MountPoint {
				logger.Warn().
					Str(constants.LogMountPoint, mount.MountPoint).
					Msg("Mount config changed, remounting...")

				if UnmountInstanceViaRcdWithFuseFallback(existing, logger) {
					untrackEndpoint(existing)
					StartMountWithRetries(instance, logger)
				}
			}
		} else {
			logger.Info().
				Str(constants.LogMountPoint, mount.MountPoint).
				Msg("New mount detected, mounting...")
			StartMountWithRetries(instance, logger)
		}
	}
}

func removeStaleMounts(conf *config.Config, logger zerolog.Logger) {
	instanceMap.Range(func(key, value interface{}) bool {
		instance := value.(*MountedEndpoint)
		if !config.IsMountInConfig(instance.MountPoint, conf) {
			logger.Warn().
				Str(constants.LogBackend, instance.BackendName).
				Msg("Mount removed from config, unmounting...")
			if UnmountInstanceViaRcdWithFuseFallback(instance, logger) {
				untrackEndpoint(instance)
			}
		}
		return true
	})
}
