package constants

// Constants for rclone
const (
	Rclone     = "rclone"
	Serve      = "serve"
	Mount      = "mount"
	MountPoint = "mountPoint="
	Addr       = "--addr"
)

// Constants for fusermount
const (
	Fusermount  = "fusermount"
	FuseUnmount = "-uz"
)

// Log constants
const (
	LogBackend    = "backend"
	LogMountPoint = "mountPoint"
	LogAddr       = "addr"
	LogProtocol   = "protocol"
	LogError      = "error"
	LogPid        = "pid"
	LogFile       = "file"
)

// Constants data files
const (
	YAMLPathEnvVar    = "RCLONE_MANAGER_CONFIG_YAML"
	DefaultYAMLPath   = "/data/config.yaml"
	RcloneConfEnvVar  = "RCLONE_MANAGER_RCLONE_CONF"
	DefaultRcloneConf = "/data/rclone.conf"
)
