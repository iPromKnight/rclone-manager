package serve_manager

import (
	"github.com/rs/zerolog"
	"os"
	"os/exec"
	"rclone-manager/internal/config"
	"sync"
	"syscall"
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
}

var (
	processMap sync.Map
)

func InitializeServeEndpoints(conf *config.Config, logger zerolog.Logger) {
	if len(conf.Serves) == 0 {
		logger.Debug().Msg("No rclone serve endpoints defined... Skipping starting any")
		return
	}

	logger.Info().Msg("Initializing all serve endpoints")
	for _, serve := range conf.Serves {
		StartServe(serve.BackendName, serve.Protocol, serve.Addr, logger)
	}

	go MonitorServeProcesses(logger)
}

func StartServe(backend, protocol, addr string, logger zerolog.Logger) *ServeProcess {
	cmd := exec.Command("rclone", "serve", protocol, backend+":", "--addr", addr)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true} // Detach from parent

	err := cmd.Start()
	if err != nil {
		logger.Error().Err(err).Str("backend", backend).Msg("Failed to start serve process")
		return nil
	}

	go func() {
		_ = cmd.Wait()
		logger.Warn().Str("backend", backend).Msgf("Process (PID: %d) exited", cmd.Process.Pid)
	}()

	serveProcess := &ServeProcess{
		PID:         cmd.Process.Pid,
		Command:     cmd,
		Backend:     backend,
		Protocol:    protocol,
		Addr:        addr,
		StartedAt:   time.Now(),
		GracePeriod: 10 * time.Second, // 10-second grace period
	}

	processMap.Store(backend, serveProcess)
	logger.Info().Int("pid", cmd.Process.Pid).Str("backend", backend).Msg("Started rclone serve process")
	return serveProcess
}

func StopServe(serveProcess *ServeProcess, logger zerolog.Logger) {
	if err := serveProcess.Command.Process.Kill(); err == nil {
		logger.Info().Int("pid", serveProcess.PID).Str("backend", serveProcess.Backend).Msg("Serve process stopped")
		processMap.Delete(serveProcess.Backend)
	} else {
		logger.Warn().Err(err).Int("pid", serveProcess.PID).Str("backend", serveProcess.Backend).Msg("Failed to stop rclone serve process")
	}
}

func Cleanup(logger zerolog.Logger) {
	logger.Info().Msg("Cleaning up all rclone serve utils")
	shouldMonitorProcesses = false
	processMap.Range(func(key, value interface{}) bool {
		serveProcess := value.(*ServeProcess)
		StopServe(serveProcess, logger)
		return true
	})
}
