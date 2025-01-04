package rclone_manager

import (
	"github.com/rs/zerolog"
	"net/http"
	"os"
	"os/exec"
	"rclone-manager/internal/config"
	"rclone-manager/internal/constants"
	"rclone-manager/internal/mount_manager"
	"rclone-manager/internal/serve_manager"
	"rclone-manager/internal/watcher"
	"syscall"
	"time"
)

func createStartRcdCommand() *exec.Cmd {
	cmd := exec.Command(constants.Rclone, constants.Rcd)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	return cmd
}

func startFileWatcher(logger zerolog.Logger) {
	filesToWatch := []string{
		constants.YAMLPath,
		constants.RcloneConf,
	}

	w := watcher.NewWatcher(func(file string) {
		reloadConfig(file, logger)
	}, logger)
	w.Watch(filesToWatch)
}

func reloadConfig(file string, logger zerolog.Logger) {
	logger.Info().Str(constants.LogFile, file).Msg("Reloading configuration...")

	conf, err := config.LoadConfig()
	if err != nil {
		logger.Error().Err(err).Msg("Failed to reload config")
		return
	}

	mount_manager.ReconcileMounts(conf, logger, &processLock)
	serve_manager.ReconcileServes(conf, logger, &processLock)

	LoadedConfig = conf

	logger.Info().Msg("Configuration reloaded successfully")
}

func pingRCD(logger zerolog.Logger) bool {
	resp, err := http.Get("http://localhost:5572")
	if err != nil || resp.StatusCode != http.StatusOK {
		return false
	}
	logger.Debug().Msg("Rclone RCD is responsive")
	return true
}

func waitForRCD(logger zerolog.Logger, maxRetries int) {
	for i := 0; i < maxRetries; i++ {
		if pingRCD(logger) {
			logger.Info().Msg("Rclone RCD is ready for mounts")
			return
		}
		logger.Warn().Msgf("Rclone RCD not ready. Retrying... (%d/%d)", i+1, maxRetries)
		time.Sleep(5 * time.Second)
	}

	logger.Fatal().Msg("Rclone RCD failed to start after retries. Exiting...")
}

func trackRCD(instance *RCloneProcess) {
	processMap.Store(constants.Rcd, instance)
}

func untrackRCD() {
	processMap.Delete(constants.Rcd)
}
