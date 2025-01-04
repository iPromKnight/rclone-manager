package constants

// Constants for rclone
const (
	Rclone     = "rclone"
	Serve      = "serve"
	Rcd        = "rcd"
	Rc         = "rc"
	Mount      = "mount/mount"
	UnmountAll = "mount/unmountall"
	Unmount    = "mount/unmount"
	MountPoint = "mountPoint="
	Fs         = "fs="
	Addr       = "--addr"
	VfsOpt     = "vfsOpt="
	MountOpt   = "mountOpt="
)

// Constants for fusermount
const (
	Fusermount  = "fusermount"
	FuseUnmount = "-u"
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
