package serve_manager

import (
	"github.com/rs/zerolog"
	"rclone-manager/internal/constants"
	"rclone-manager/internal/utils"
	"time"
)

var (
	shouldMonitorProcesses bool
)

func MonitorServeProcesses(logger zerolog.Logger) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error().Msgf("MonitorProcesses crashed: %v", r)
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
			serveProcess := value.(*ServeProcess)

			if time.Since(serveProcess.StartedAt) < serveProcess.GracePeriod {
				logger.Debug().Int(constants.LogPid, serveProcess.PID).Msg("Skipping process check (within grace period)")
				return true
			}

			if !utils.ProcessIsRunning(serveProcess.PID) {
				logger.Warn().Str(constants.LogBackend, serveProcess.Backend).
					Msgf("Process (PID: %d) died. Restarting...", serveProcess.PID)

				newServe := &ServeProcess{
					Backend:  serveProcess.Backend,
					Protocol: serveProcess.Protocol,
					Addr:     serveProcess.Addr,
				}

				newProcess := StartServeWithRetries(newServe, logger)

				if newProcess != nil {
					trackServe(newProcess)
					logger.Info().Str(constants.LogBackend, serveProcess.Backend).
						Msgf("Successfully restarted serve with new PID: %d", newProcess.PID)
				} else {
					logger.Error().Str(constants.LogBackend, serveProcess.Backend).
						Msg("Failed to restart serve process")
				}
			}
			return true
		})
		time.Sleep(10 * time.Second)
	}
}
