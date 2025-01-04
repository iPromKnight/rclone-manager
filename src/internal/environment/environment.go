package environment

import (
	"fmt"
	"os"
	"strings"
)

func PrepareEnvironment(envVars map[string]string) []string {
	if envVars == nil {
		return os.Environ()
	}

	envMap := make(map[string]string)

	for _, env := range os.Environ() {
		parts := splitEnv(env)
		envMap[parts[0]] = parts[1]
	}

	for key, value := range envVars {
		envMap[key] = value
	}

	return mapToSlice(envMap)
}

func splitEnv(env string) []string {
	parts := make([]string, 2)
	idx := strings.Index(env, "=")
	if idx != -1 {
		parts[0] = env[:idx]
		parts[1] = env[idx+1:]
	} else {
		parts[0] = env
		parts[1] = ""
	}
	return parts
}

func mapToSlice(envMap map[string]string) []string {
	var env []string
	for k, v := range envMap {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	return env
}
