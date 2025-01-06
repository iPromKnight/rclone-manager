package main

import (
	"github.com/rs/zerolog"
	"os"
	"os/signal"
	"rclone-manager/internal/rclone_manager"
	"strings"
	"syscall"
	"time"
)

var logger zerolog.Logger

func init() {
	debugMode := os.Getenv("DEBUG_MODE")
	if strings.ToLower(debugMode) == "true" || strings.ToLower(debugMode) == "1" {
		logger = zerolog.New(os.Stdout).Level(zerolog.DebugLevel).With().Timestamp().Logger()
	} else {
		logger = zerolog.New(os.Stdout).Level(zerolog.InfoLevel).With().Timestamp().Logger()
	}
}

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	rclone_manager.InitializeRClone(logger)

	for {
		select {
		case sig := <-sigs:
			logger.Warn().Msgf("Received signal %v, shutting down...", sig)
			rclone_manager.StopRclone(logger)
			os.Exit(0)
		default:
			time.Sleep(1 * time.Second)
		}
	}
}
