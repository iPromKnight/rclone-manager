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
	YAMLPath   = "/data/config.yaml"
	RcloneConf = "/data/rclone.conf"
)
