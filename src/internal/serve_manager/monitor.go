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
		logger.Debug().Msg("Checking serve processes...")
		if !shouldMonitorProcesses {
			logger.Info().Msg("Stopping rclone serve process monitor")
			break
		}
		tracker.Range(func(key, value interface{}) bool {
			serveProcess := value.(*ServeProcess)

			if !shouldMonitorProcesses {
				return false
			}

			if time.Since(serveProcess.StartedAt) < serveProcess.GracePeriod {
				logger.Debug().Int(constants.LogPid, serveProcess.PID).Msg("Skipping process check (within grace period)")
				return true
			}

			if !utils.ProcessIsRunning(serveProcess.PID) {
				logger.Warn().Str(constants.LogBackend, serveProcess.BackendName).
					Msgf("Process (PID: %d) died. Restarting...", serveProcess.PID)

				if !shouldMonitorProcesses {
					return false
				}

				newServe := &ServeProcess{
					Protocol:      serveProcess.Protocol,
					Addr:          serveProcess.Addr,
					RcloneProcess: serveProcess.RcloneProcess,
				}

				newProcess := StartServeWithRetries(newServe, logger)

				if !shouldMonitorProcesses {
					return false
				}

				if newProcess != nil {
					tracker.Track(newProcess.BackendName, newProcess)
					logger.Info().Str(constants.LogBackend, serveProcess.BackendName).
						Msgf("Successfully restarted serve with new PID: %d", newProcess.PID)
				} else {
					logger.Error().Str(constants.LogBackend, serveProcess.BackendName).
						Msg("Failed to restart serve process")
				}
			}
			logger.Debug().Str("Backend", serveProcess.BackendName).Msg("Serve is fine. Nothing to do.")
			return true
		})
		time.Sleep(10 * time.Second)
	}
}
