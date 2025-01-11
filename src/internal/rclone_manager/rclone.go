package rclone_manager

import (
	"github.com/rs/zerolog"
	"rclone-manager/internal/config"
	"rclone-manager/internal/constants"
	"rclone-manager/internal/environment"
	"rclone-manager/internal/mount_manager"
	"rclone-manager/internal/serve_manager"
	"rclone-manager/internal/watcher"
	"sync"
)

var (
	LoadedConfig *config.Config
	processLock  sync.Mutex
)

func InitializeRClone(logger zerolog.Logger) {

	conf, err := config.LoadConfig()
	if err != nil {

		logger.Fatal().Err(err).Msg("Failed to load configuration")
	}

	if len(conf.Serves) == 0 && len(conf.Mounts) == 0 {
		logger.Warn().Msg("No serves or mounts found in configuration. Exiting...")
		return
	}

	processLock.Lock()
	defer processLock.Unlock()

	LoadedConfig = conf

	if len(conf.Serves) > 0 {
		go serve_manager.InitializeServeEndpoints(conf, logger, &processLock)
	}

	if len(conf.Mounts) > 0 {
		go mount_manager.InitializeMountEndpoints(conf, logger, &processLock)
	}

	yamlPath := environment.GetEnvWithFallback(constants.YAMLPathEnvVar, constants.DefaultYAMLPath)
	rcloneConfPath := environment.GetEnvWithFallback(constants.RcloneConfEnvVar, constants.DefaultRcloneConf)

	filesToWatch := []string{
		yamlPath,
		rcloneConfPath,
	}

	watcher.StartNewFileWatcher(filesToWatch, reloadConfig, logger)
}

func StopRclone(logger zerolog.Logger) {
	processLock.Lock()
	defer processLock.Unlock()

	if LoadedConfig != nil {
		serve_manager.Cleanup(logger)
		mount_manager.Cleanup(LoadedConfig, logger)
	}
}
