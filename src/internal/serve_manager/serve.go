package serve_manager

import (
	"github.com/rs/zerolog"
	"os/exec"
	"rclone-manager/internal/config"
	"rclone-manager/internal/constants"
	"sync"
	"time"
)

type ServeProcess struct {
	PID         int
	Command     *exec.Cmd
	Backend     string
	Protocol    string
	Addr        string
	StartedAt   time.Time
	GracePeriod time.Duration
	EnvVars     map[string]string
}

var (
	processMap    sync.Map
	currentRCDEnv map[string]interface{}
)

func SetRCDEnv(env map[string]interface{}) {
	currentRCDEnv = env
}

func InitializeServeEndpoints(conf *config.Config, logger zerolog.Logger, processLock *sync.Mutex) {
	processLock.Lock()
	defer processLock.Unlock()

	if len(conf.Serves) == 0 {
		logger.Debug().Msg("No rclone serve endpoints defined... Skipping starting any")
		return
	}

	logger.Info().Msg("Initializing all serve endpoints")
	for _, serve := range conf.Serves {
		instance := &ServeProcess{
			Backend:  serve.BackendName,
			Protocol: serve.Protocol,
			Addr:     serve.Addr,
			EnvVars:  serve.Environment,
		}
		StartServeWithRetries(instance, logger)
	}

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
				Str(constants.LogBackend, instance.Backend).
				Str(constants.LogProtocol, instance.Protocol).
				Str(constants.LogAddr, instance.Addr).
				Msg("Serve started successfully.")
			instance.PID = cmd.Process.Pid
			instance.StartedAt = time.Now()
			instance.GracePeriod = 10 * time.Second
			trackServe(instance)
			return instance
		}
		logger.Warn().AnErr(constants.LogError, err).Msgf("Serve failed. Retrying %d/3...", retries+1)
		retries++
		time.Sleep(5 * time.Second)
	}
	logger.Error().Str(constants.LogBackend, instance.Backend).Msg("Failed to start serve after 3 attempts.")
	return nil
}

func StopServe(instance *ServeProcess, logger zerolog.Logger) {
	logger.Info().Str(constants.LogBackend, instance.Backend).Msg("Stopping serve process...")
	if err := instance.Command.Process.Kill(); err == nil {
		untrackServe(instance)
		logger.Info().Int(constants.LogPid, instance.PID).Str(constants.LogBackend, instance.Backend).Msg("Serve process stopped")
	} else {
		logger.Warn().AnErr(constants.LogError, err).Int(constants.LogPid, instance.PID).Str(constants.LogBackend, instance.Backend).Msg("Failed to stop serve process")
	}
}

func Cleanup(logger zerolog.Logger) {
	logger.Info().Msg("Cleaning up all rclone serve processes")
	processMap.Range(func(key, value interface{}) bool {
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
