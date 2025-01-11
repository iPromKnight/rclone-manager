package config

import (
	"gopkg.in/yaml.v3"
	"os"
	"rclone-manager/internal/constants"
	"rclone-manager/internal/environment"
)

type Config struct {
	Serves []struct {
		BackendName string            `yaml:"backendName"`
		Protocol    string            `yaml:"protocol"`
		Addr        string            `yaml:"addr"`
		Environment map[string]string `yaml:"environment,omitempty"`
	} `yaml:"serves"`

	Mounts []struct {
		BackendName string            `yaml:"backendName"`
		MountPoint  string            `yaml:"mountPoint"`
		Environment map[string]string `yaml:"environment,omitempty"`
	} `yaml:"mounts"`
}

func LoadConfig() (*Config, error) {
	yamlPath := environment.GetEnvWithFallback(constants.YAMLPathEnvVar, constants.DefaultYAMLPath)
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		return nil, err
	}

	var config Config
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

func IsMountInConfig(mountPoint string, conf *Config) bool {
	for _, mount := range conf.Mounts {
		if mount.MountPoint == mountPoint {
			return true
		}
	}
	return false
}

func IsServeInConfig(backend string, conf *Config) bool {
	for _, mount := range conf.Serves {
		if mount.BackendName == backend {
			return true
		}
	}
	return false
}
