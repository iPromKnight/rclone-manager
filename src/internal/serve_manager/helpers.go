package serve_manager

import (
	"fmt"
	"github.com/rs/zerolog"
	"os"
	"os/exec"
	"rclone-manager/internal/config"
	"rclone-manager/internal/constants"
	"rclone-manager/internal/environment"
	"rclone-manager/internal/instance_tracker"
)

func createServeCommand(instance *ServeProcess) *exec.Cmd {
	backendArg := fmt.Sprintf("%s:", instance.BackendName)

	cmd := exec.Command(
		constants.Rclone, constants.Serve, instance.Protocol, backendArg, constants.Addr, instance.Addr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Env = environment.PrepareEnvironment(instance.Environment)

	return cmd
}

func setupServesFromConfig(conf *config.Config, logger zerolog.Logger) {
	for _, serve := range conf.Serves {
		instance := &ServeProcess{
			Protocol: serve.Protocol,
			Addr:     serve.Addr,
			RcloneProcess: instance_tracker.RcloneProcess{
				BackendName: serve.BackendName,
				Environment: serve.Environment,
			},
		}
		if existing, ok := tracker.Get(serve.BackendName); ok {
			if existing.Protocol != serve.Protocol || existing.Addr != serve.Addr {
				logger.Warn().
					Str(constants.LogBackend, serve.BackendName).
					Msg("Serve config changed, restarting...")
				StopServe(existing, logger)
				StartServeWithRetries(instance, logger)
			}
		} else {
			logger.Info().
				Str(constants.LogBackend, serve.BackendName).
				Msg("New serve detected, starting...")
			StartServeWithRetries(instance, logger)
		}
	}
}

func removeStaleServes(conf *config.Config, logger zerolog.Logger) {
	var staleKeys []interface{}

	tracker.Range(func(key, value interface{}) bool {
		instance := value.(*ServeProcess)
		if !config.IsServeInConfig(instance.BackendName, conf) {
			logger.Warn().
				Str(constants.LogBackend, instance.BackendName).
				Msg("Serve removed from config, stopping...")
			staleKeys = append(staleKeys, key)
		}
		return true
	})

	for _, key := range staleKeys {
		if instance, ok := tracker.Get(key.(string)); ok {
			StopServe(instance, logger)
		}
	}
}
