package serve_manager

import (
	"fmt"
	"github.com/rs/zerolog"
	"os"
	"os/exec"
	"rclone-manager/internal/config"
	"rclone-manager/internal/constants"
)

func getServeInstance(key string) (*ServeProcess, bool) {
	if val, ok := processMap.Load(key); ok {
		instance, valid := val.(*ServeProcess)
		return instance, valid
	}
	return nil, false
}

func createServeCommand(instance *ServeProcess) *exec.Cmd {
	backendArg := fmt.Sprintf("%s:", instance.Backend)

	cmd := exec.Command(
		constants.Rclone, constants.Serve, instance.Protocol, backendArg, constants.Addr, instance.Addr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd
}

func trackServe(instance *ServeProcess) {
	processMap.Store(instance.Backend, instance)
}

func untrackServe(instance *ServeProcess) {
	processMap.Delete(instance.Backend)
}

func setupServesFromConfig(conf *config.Config, logger zerolog.Logger) {
	for _, serve := range conf.Serves {
		instance := &ServeProcess{
			Backend:  serve.BackendName,
			Protocol: serve.Protocol,
			Addr:     serve.Addr,
		}
		if existing, ok := getServeInstance(serve.BackendName); ok {
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

	processMap.Range(func(key, value interface{}) bool {
		instance := value.(*ServeProcess)
		if !config.IsServeInConfig(instance.Backend, conf) {
			logger.Warn().
				Str(constants.LogBackend, instance.Backend).
				Msg("Serve removed from config, stopping...")
			staleKeys = append(staleKeys, key)
		}
		return true
	})

	for _, key := range staleKeys {
		if instance, ok := getServeInstance(key.(string)); ok {
			StopServe(instance, logger)
		}
	}
}
