package rclone_manager

import (
	"github.com/rs/zerolog"
	"rclone-manager/internal/config"
	"rclone-manager/internal/constants"
	"rclone-manager/internal/mount_manager"
	"rclone-manager/internal/utils"
	"time"
)

var (
	shouldMonitorProcesses bool
)

func MonitorRCDProcess(conf *config.Config, logger zerolog.Logger) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error().Msgf("MonitorRCD crashed: %v", r)
		}
	}()

	logger.Info().Msg("Starting rclone serve process monitor...")
	shouldMonitorProcesses = true

	for {
		if !shouldMonitorProcesses {
			logger.Info().Msg("Stopping rclone serve process monitor")
			break
		}
		processMap.Range(func(key, value interface{}) bool {
			rCloneProcess := value.(*RCloneProcess)

			if time.Since(rCloneProcess.StartedAt) < rCloneProcess.GracePeriod {
				logger.Debug().Int(constants.LogPid, rCloneProcess.PID).Msg("Skipping process check (within grace period)")
				return true
			}

			if !utils.ProcessIsRunning(rCloneProcess.PID) {
				logger.Warn().Msgf("Process (PID: %d) died. Restarting...", rCloneProcess.PID)
				mount_manager.UnmountAllByPath(conf, logger)
				newProcess := StartRcloneRemoteDaemon(logger)

				if newProcess != nil {
					processMap.Store(key, newProcess)
					logger.Info().Msgf("Successfully restarted rclone in RCD mode with new PID: %d", newProcess.PID)
				} else {
					logger.Error().Msg("Failed to restart rclone RCD process")
				}
			}
			return true
		})
		time.Sleep(10 * time.Second)
	}
}
