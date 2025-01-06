package rclone_manager

import (
	"github.com/rs/zerolog"
	"rclone-manager/internal/config"
	"rclone-manager/internal/constants"
	"rclone-manager/internal/mount_manager"
	"rclone-manager/internal/serve_manager"
)

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
