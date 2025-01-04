package main

import (
	"github.com/rs/zerolog"
	"os"
	"os/signal"
	"rclone-manager/internal/rclone_manager"
	"syscall"
	"time"
)

var logger = zerolog.New(os.Stdout).With().Timestamp().Logger()

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	rclone_manager.InitializeRCD(logger)

	for {
		select {
		case sig := <-sigs:
			logger.Warn().Msgf("Received signal %v, shutting down...", sig)
			rclone_manager.StopRcloneRemoteDaemon(logger)
			os.Exit(0)
		default:
			time.Sleep(1 * time.Second)
		}
	}
}
