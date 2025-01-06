package mount_manager

import (
	"github.com/rs/zerolog"
	"rclone-manager/internal/constants"
	"rclone-manager/internal/utils"
	"time"
)

var (
	shouldMonitorProcesses bool
)

func MonitorMountProcesses(logger zerolog.Logger) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error().Msgf("MonitorProcesses crashed: %v", r)
		}
	}()

	logger.Info().Msg("Starting rclone mount process monitor...")
	shouldMonitorProcesses = true

	for {
		logger.Debug().Msg("Checking mount processes...")
		if !shouldMonitorProcesses {
			logger.Info().Msg("Stopping rclone mount process monitor")
			break
		}
		tracker.Range(func(key, value interface{}) bool {
			if !shouldMonitorProcesses {
				return false
			}

			mountProcess := value.(*MountProcess)

			if time.Since(mountProcess.StartedAt) < mountProcess.GracePeriod {
				logger.Debug().Int(constants.LogPid, mountProcess.PID).Msg("Skipping process check (within grace period)")
				return true
			}

			if !utils.ProcessIsRunning(mountProcess.PID) {
				logger.Warn().Str(constants.LogBackend, mountProcess.BackendName).
					Msgf("Process (PID: %d) died. Restarting...", mountProcess.PID)

				if !shouldMonitorProcesses {
					return false
				}

				UnmountEndpoint(mountProcess, logger)

				if !shouldMonitorProcesses {
					return false
				}

				newMount := &MountProcess{
					MountPoint:    mountProcess.MountPoint,
					RcloneProcess: mountProcess.RcloneProcess,
				}

				newProcess := StartMountWithRetries(newMount, logger)

				if !shouldMonitorProcesses {
					return false
				}

				if newProcess != nil {
					tracker.Track(newProcess.BackendName, newProcess)
					logger.Info().Str(constants.LogBackend, mountProcess.BackendName).
						Msgf("Successfully restarted mount with new PID: %d", newProcess.PID)
				} else {
					logger.Error().Str(constants.LogBackend, mountProcess.BackendName).
						Msg("Failed to restart mount process")
				}
			}

			logger.Debug().Str("MountPoint", mountProcess.MountPoint).Msg("Mount is mounted fine. Nothing to do.")
			return true
		})
		time.Sleep(10 * time.Second)
	}
}
