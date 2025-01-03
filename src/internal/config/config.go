package config

import (
	"gopkg.in/yaml.v3"
	"os"
)

const YAMLPath = "/data/config.yaml"

type Config struct {
	Serves []struct {
		BackendName string `yaml:"backendName"`
		Protocol    string `yaml:"protocol"`
		Addr        string `yaml:"addr"`
	} `yaml:"serves"`

	Mounts []struct {
		BackendName string `yaml:"backendName"`
		MountPoint  string `yaml:"mountPoint"`
	} `yaml:"mounts"`
}

func LoadConfig() (*Config, error) {
	data, err := os.ReadFile(YAMLPath)
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
