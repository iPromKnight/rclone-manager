package mount_manager

import (
	"github.com/rs/zerolog"
	"os"
	"os/exec"
	"rclone-manager/internal/config"
	"syscall"
	"time"
)

func InitializeMounts(conf *config.Config, logger zerolog.Logger) {
	if len(conf.Mounts) == 0 {
		logger.Debug().Msg("No rclone mount endpoints defined... Skipping...")
		return
	}

	logger.Info().Msg("Initializing all Mounts")
	for _, mount := range conf.Mounts {
		StartMountWithRetries(mount.BackendName, mount.MountPoint, logger)
	}
}

func StartMountWithRetries(backend string, mountPoint string, logger zerolog.Logger) {
	retries := 0
	for retries < 3 {
		ensureMountPointExists(mountPoint, logger)
		cmd := exec.Command("rclone", "rc", "mount/mount", "fs="+backend+":", "mountPoint="+mountPoint)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err == nil {
			logger.Info().Str("backend", backend).Str("mountPoint", mountPoint).
				Msg("Mount successful.")
			return
		}
		logger.Warn().Err(err).Msgf("Mount failed. Retrying %d/3...", retries+1)
		retries++
		time.Sleep(5 * time.Second)
	}
	logger.Error().Str("backend", backend).Msg("Failed to mount after 3 attempts.")
}

func StopAllMountsViaRCD(logger zerolog.Logger) {
	logger.Info().Msg("Unmounting all rclone mounts")
	cmd := exec.Command("rclone", "rc", "mount/umountall")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	err := cmd.Start()
	if err != nil {
		logger.Error().AnErr("error", err).Msg("Failed to unmount all rclone mounts")
		return
	}

	logger.Info().Msg("Unmounted all rclone mounts successfully.")
}

func UnmountAllByPath(conf *config.Config, logger zerolog.Logger) {
	logger.Info().Msg("Unmounting all paths listed in config...")

	for _, mount := range conf.Mounts {
		logger.Info().Str("path", mount.MountPoint).Msg("Unmounting...")
		cmd := exec.Command("fusermount", "-u", mount.MountPoint)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			logger.Warn().Err(err).Str("path", mount.MountPoint).Msg("Failed to unmount path. It may not be mounted.")
		} else {
			logger.Info().Str("path", mount.MountPoint).Msg("Unmounted successfully.")
		}
	}
}

func ReloadMounts(conf *config.Config, logger zerolog.Logger) {
	UnmountAllByPath(conf, logger)
	logger.Info().Msg("Reloading all mounts from config...")

	for _, mount := range conf.Mounts {
		StartMountWithRetries(mount.BackendName, mount.MountPoint, logger)
	}
}

func ensureMountPointExists(mountPoint string, logger zerolog.Logger) {
	if _, err := os.Stat(mountPoint); os.IsNotExist(err) {
		logger.Info().Str("mountPoint", mountPoint).Msg("Creating mount point...")
		err := os.MkdirAll(mountPoint, 0777)
		if err != nil {
			logger.Error().Err(err).Str("mountPoint", mountPoint).Msg("Failed to create mount point")
		} else {
			logger.Info().Str("mountPoint", mountPoint).Msg("Mount point created successfully.")
		}
	}
}
