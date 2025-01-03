package serve_manager

import (
	"github.com/rs/zerolog"
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
				logger.Debug().Int("pid", serveProcess.PID).Msg("Skipping process check (within grace period)")
				return true
			}

			if !utils.ProcessIsRunning(serveProcess.PID) {
				logger.Warn().Str("backend", serveProcess.Backend).
					Msgf("Process (PID: %d) died. Restarting...", serveProcess.PID)

				newProcess := StartServe(
					serveProcess.Backend,
					serveProcess.Protocol,
					serveProcess.Addr,
					logger,
				)

				if newProcess != nil {
					processMap.Store(key, newProcess)
					logger.Info().Str("backend", serveProcess.Backend).
						Msgf("Successfully restarted serve with new PID: %d", newProcess.PID)
				} else {
					logger.Error().Str("backend", serveProcess.Backend).
						Msg("Failed to restart serve process")
				}
			}
			return true
		})
		time.Sleep(10 * time.Second)
	}
}
