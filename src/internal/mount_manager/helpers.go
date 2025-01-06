package mount_manager

import (
	"encoding/json"
	"fmt"
	"github.com/rclone/rclone/cmd/mountlib"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/vfs/vfscommon"
	"github.com/rs/zerolog"
	"os"
	"os/exec"
	"rclone-manager/internal/config"
	"rclone-manager/internal/constants"
	"rclone-manager/internal/environment"
	"reflect"
	"strconv"
	"strings"
	"syscall"
)

type mountPayloads struct {
	mountOptPayload string
	vfsOptPayload   string
}

func trackEndpoint(instance *MountedEndpoint) {
	instanceMap.Store(instance.BackendName, instance)
}

func untrackEndpoint(instance *MountedEndpoint) {
	instanceMap.Delete(instance.BackendName)
}

func getMountedEndpoint(key interface{}) (*MountedEndpoint, bool) {
	if val, ok := instanceMap.Load(key); ok {
		instance, valid := val.(*MountedEndpoint)
		return instance, valid
	}
	return nil, false
}

func createMountCommand(instance *MountedEndpoint, logger zerolog.Logger) *exec.Cmd {
	fsArg := fmt.Sprintf("%s%s:", constants.Fs, instance.BackendName)
	mountPointArg := fmt.Sprintf("%s%s", constants.MountPoint, instance.MountPoint)

	payloads, err := constructPayloadsFromCombinedEnvVars(currentRCDEnv, instance.Environment, logger)
	if err != nil {
		logger.Error().AnErr(constants.LogError, err).Str(constants.LogBackend, instance.BackendName).
			Msg("Failed to construct mount payloads")
		return nil
	}

	mountOptArg := fmt.Sprintf("%s%s", constants.MountOpt, payloads.mountOptPayload)
	vfsArg := fmt.Sprintf("%s%s", constants.VfsOpt, payloads.vfsOptPayload)

	cmd := exec.Command(constants.Rclone, constants.Rc, constants.Mount, fsArg, mountPointArg, vfsArg, mountOptArg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	cmd.Env = environment.PrepareEnvironment(instance.Environment)

	return cmd
}

func constructPayloadsFromCombinedEnvVars(options map[string]interface{}, envVars map[string]string, logger zerolog.Logger) (*mountPayloads, error) {
	logger.Debug().Interface("options", options).Msg("Fetched rclone options")

	vfsOptions, vfsExists := options["vfs"].(map[string]interface{})
	mountOptions, mountExists := options["mount"].(map[string]interface{})

	if !vfsExists || !mountExists {
		return nil, fmt.Errorf("missing vfs or mount options in fetched rclone config")
	}

	// Extract config tags from struct
	mountTags := extractConfigTags(&mountlib.Options{}, logger)
	vfsTags := extractConfigTags(&vfscommon.Options{}, logger)

	logger.Debug().Interface("envVars", envVars).Msg("Augmenting options with environment variables")

	vfsEnvs := envVars
	mountEnvs := envVars

	// Update options using extracted config tags
	updateOptionsWithEnv(vfsOptions, vfsEnvs, "vfs", vfsTags, logger)
	logger.Debug().Interface("vfsOptions", vfsOptions).Msg("Updated VFS Options")

	updateOptionsWithEnv(mountOptions, mountEnvs, "mount", mountTags, logger)
	logger.Debug().Interface("mountOptions", mountOptions).Msg("Updated Mount Options")

	vfsPayloadJson, err := json.Marshal(vfsOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize vfs options: %w", err)
	}

	mountPayloadJson, err := json.Marshal(mountOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize mount options: %w", err)
	}

	return &mountPayloads{
		vfsOptPayload:   string(vfsPayloadJson),
		mountOptPayload: string(mountPayloadJson),
	}, nil
}

func extractConfigTags(opt interface{}, logger zerolog.Logger) map[string]string {
	tags := make(map[string]string)
	t := reflect.TypeOf(opt).Elem()

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if tag, ok := field.Tag.Lookup("config"); ok {
			// Normalize extracted tag (replace underscores with hyphens)
			normalizedTag := strings.ToLower(strings.ReplaceAll(tag, "_", "-"))
			tags[normalizedTag] = field.Name
		}
	}

	logger.Debug().Interface("configTags", tags).Msg("Extracted config tags")

	return tags
}

func updateOptionsWithEnv(options map[string]interface{}, envVars map[string]string, section string, configTags map[string]string, logger zerolog.Logger) {
	logger.Debug().Str("section", section).Interface("options", options).Msg("Updating options from environment variables")

	for key, value := range envVars {
		if strings.HasPrefix(key, fmt.Sprintf("RCLONE_%s_", strings.ToUpper(section))) {
			cleanKey := strings.TrimPrefix(key, fmt.Sprintf("RCLONE_%s_", strings.ToUpper(section)))
			cleanKey = strings.ToLower(strings.ReplaceAll(cleanKey, "_", "-"))

			// Retain vfs_ prefix if present
			if section == "vfs" && !strings.HasPrefix(cleanKey, "vfs-") {
				cleanKey = "vfs-" + cleanKey
			}

			fieldName, exists := configTags[cleanKey]
			if exists {
				options[fieldName] = tryConvertType(options[fieldName], value, logger)
				logger.Debug().
					Str("field", fieldName).
					Str("value", value).
					Msg("Updated option from environment variable")
			} else {
				logger.Warn().
					Str("field", cleanKey).
					Msg("No matching config tag found")
			}
		}
	}
}

func tryConvertType(currentValue interface{}, newValue string, logger zerolog.Logger) interface{} {
	switch v := currentValue.(type) {
	case fs.SizeSuffix:
		// Direct conversion for SizeSuffix
		var size fs.SizeSuffix
		err := size.Set(newValue)
		if err != nil {
			logger.Warn().
				Str("type", "fs.SizeSuffix").
				Str("value", newValue).
				Msg("Failed to convert value to SizeSuffix")
			return currentValue
		}
		return size
	case float64: // This handles unmarshalled JSON numbers as float64
		// Attempt to convert to fs.SizeSuffix if it's expected to be a size
		var size fs.SizeSuffix
		err := size.Set(newValue)
		if err == nil {
			return size
		}
		// Fallback to int if conversion fails
		return int64(v)
	case int, int64:
		parsed, _ := strconv.ParseInt(newValue, 10, 64)
		return parsed
	case bool:
		parsed, _ := strconv.ParseBool(newValue)
		return parsed
	case string:
		return newValue
	default:
		logger.Warn().
			Str("type", reflect.TypeOf(v).String()).
			Msg("Unsupported conversion type")
		return newValue
	}
}

func createUnmountCommand(instance *MountedEndpoint) *exec.Cmd {
	mountPointArg := fmt.Sprintf("%s%s", constants.MountPoint, instance.MountPoint)

	cmd := exec.Command(constants.Rclone, constants.Rc, constants.Unmount, mountPointArg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd
}

func createUnmountAllCommand() *exec.Cmd {
	cmd := exec.Command(constants.Rclone, constants.Rc, constants.UnmountAll)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd
}

func createFuseUnmountCommand(instance *MountedEndpoint) *exec.Cmd {
	cmd := exec.Command(constants.Fusermount, constants.FuseUnmount, instance.MountPoint)
	cmd.Stdout = os.Stdout
	return cmd
}

func ensureMountPointExists(mountPoint string, logger zerolog.Logger) {
	if _, err := os.Stat(mountPoint); os.IsNotExist(err) {
		logger.Info().Str(constants.LogMountPoint, mountPoint).Msg("Creating mount point...")
		err := os.MkdirAll(mountPoint, 0777)
		if err != nil {
			logger.Error().Err(err).Str(constants.LogMountPoint, mountPoint).
				Msg("Failed to create mount point")
		} else {
			logger.Info().Str(constants.LogMountPoint, mountPoint).
				Msg("Mount point created successfully.")
		}
	}
}

func setupMountsFromConfig(conf *config.Config, logger zerolog.Logger) {
	for _, mount := range conf.Mounts {
		instance := &MountedEndpoint{
			BackendName: mount.BackendName,
			MountPoint:  mount.MountPoint,
			Environment: mount.Environment,
		}
		if existing, ok := getMountedEndpoint(mount.BackendName); ok {
			if existing.MountPoint != instance.MountPoint {
				logger.Warn().
					Str(constants.LogMountPoint, mount.MountPoint).
					Msg("Mount config changed, remounting...")

				if UnmountInstanceViaRcdWithFuseFallback(existing, logger) {
					untrackEndpoint(existing)
					StartMountWithRetries(instance, logger)
				}
			}
		} else {
			logger.Info().
				Str(constants.LogMountPoint, mount.MountPoint).
				Msg("New mount detected, mounting...")
			StartMountWithRetries(instance, logger)
		}
	}
}

func removeStaleMounts(conf *config.Config, logger zerolog.Logger) {
	instanceMap.Range(func(key, value interface{}) bool {
		instance := value.(*MountedEndpoint)
		if !config.IsMountInConfig(instance.MountPoint, conf) {
			logger.Warn().
				Str(constants.LogBackend, instance.BackendName).
				Msg("Mount removed from config, unmounting...")
			if UnmountInstanceViaRcdWithFuseFallback(instance, logger) {
				untrackEndpoint(instance)
			}
		}
		return true
	})
}
