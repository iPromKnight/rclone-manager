package serve_manager

import (
	"github.com/rs/zerolog"
	"rclone-manager/internal/config"
	"rclone-manager/internal/constants"
	"rclone-manager/internal/instance_tracker"
	"sync"
	"time"
)

type ServeProcess struct {
	instance_tracker.RcloneProcess
	Protocol string
	Addr     string
}

var tracker instance_tracker.InstanceTracker[ServeProcess]

func InitializeServeEndpoints(conf *config.Config, logger zerolog.Logger, processLock *sync.Mutex) {
	processLock.Lock()
	defer processLock.Unlock()

	if len(conf.Serves) == 0 {
		logger.Debug().Msg("No rclone serve endpoints defined... Skipping starting any")
		return
	}

	logger.Info().Msg("Initializing all serve endpoints")
	setupServesFromConfig(conf, logger)

	go MonitorServeProcesses(logger)
}

func StartServeWithRetries(instance *ServeProcess, logger zerolog.Logger) *ServeProcess {
	retries := 0
	for retries < 3 {
		cmd := createServeCommand(instance)
		instance.Command = cmd
		err := cmd.Start()
		if err == nil {
			logger.Info().
				Str(constants.LogBackend, instance.BackendName).
				Str(constants.LogProtocol, instance.Protocol).
				Str(constants.LogAddr, instance.Addr).
				Msg("Serve started successfully.")
			instance.PID = cmd.Process.Pid
			instance.StartedAt = time.Now()
			instance.GracePeriod = 10 * time.Second
			tracker.Track(instance.BackendName, instance)
			return instance
		}
		logger.Warn().AnErr(constants.LogError, err).Msgf("Serve failed. Retrying %d/3...", retries+1)
		retries++
		time.Sleep(5 * time.Second)
	}
	logger.Error().Str(constants.LogBackend, instance.BackendName).Msg("Failed to start serve after 3 attempts.")
	return nil
}

func StopServe(instance *ServeProcess, logger zerolog.Logger) {
	logger.Info().Str(constants.LogBackend, instance.BackendName).Msg("Stopping serve process...")
	if err := instance.Command.Process.Kill(); err == nil {
		tracker.Untrack(instance.BackendName)
		logger.Info().Int(constants.LogPid, instance.PID).Str(constants.LogBackend, instance.BackendName).Msg("Serve process stopped")
	} else {
		logger.Warn().AnErr(constants.LogError, err).Int(constants.LogPid, instance.PID).Str(constants.LogBackend, instance.BackendName).Msg("Failed to stop serve process")
	}
}

func Cleanup(logger zerolog.Logger) {
	logger.Info().Msg("Cleaning up all rclone serve processes")
	tracker.Range(func(key, value interface{}) bool {
		instance := value.(*ServeProcess)
		StopServe(instance, logger)
		return true
	})
}

func ReconcileServes(conf *config.Config, logger zerolog.Logger, processLock *sync.Mutex) {
	processLock.Lock()
	defer processLock.Unlock()

	logger.Info().Msg("Reconciling serves...")

	setupServesFromConfig(conf, logger)
	removeStaleServes(conf, logger)
}
