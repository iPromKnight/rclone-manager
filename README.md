# Rclone Manager

## Overview
Rclone Manager is a containerized service designed to manage and monitor Rclone remote control daemon (RCD) operations. It provides automated handling of Rclone mounts and serve endpoints, ensuring persistent remote filesystem mounts and exposing remote directories over protocols such as WebDAV.

The application allows full configuration of Rclone via environment variables, providing flexibility and ease of management. It uses Docker to create isolated, scalable deployments that simplify mount handling and remote serving.

---

## Why Rclone Manager Exists
Managing multiple Rclone mounts and serve endpoints manually can be cumbersome, especially in large environments with multiple remote storage backends. This application automates:

- **Mounting remote storage** using Rclone.
- **Serving directories** via protocols like WebDAV.
- **Monitoring and restarting RCD** in case of crashes.
- **Graceful shutdown and unmounting** during container stop events.

The goal is to create a reliable, self-healing solution for Rclone operations that minimizes downtime and administrative overhead.
Not only this, but rather than spawning separate rclone processes for each mount, it centralizes them through the RCD API, reducing resource consumption and complexity.
This allows for the added bonus of having a shared bwlimit across all mounts, and a single point of control for all rclone operations.

---

## How It Works
### Mounts
- Rclone Manager reads from a configuration file (`config.yaml`) to mount specified remote storage backends at designated paths.
- If the RCD process dies or restarts, mounts are re-established automatically.
- Directories are created automatically if they don't exist.

### Serve Endpoints
- The application can also expose remote directories over protocols such as WebDAV.
- Serve processes are monitored separately from RCD to ensure continued operation, even if RCD restarts.

### RCD Monitoring
- The app continuously monitors the RCD process.
- If RCD dies unexpectedly, it is restarted with a grace period to avoid unnecessary restarts.
- All mounts are unmounted and remounted to ensure no stale mounts persist.

---

## Docker Compose Configuration
### Compose File (`compose.yml`)
```yaml
services:
  rclone-manager:
    build:
      context: src
      dockerfile: Dockerfile
    image: ipromknight/rclone-manager:latest
    container_name: rclone-manager
    restart: unless-stopped
    stop_signal: SIGTERM
    stop_grace_period: 30s
    ports:
      - "5572:5572"  # Rclone RC API
      - "8080:8080"  # WebDAV Serve Example
    devices:
      - /dev/fuse:/dev/fuse:rwm
    cap_add:
      - SYS_ADMIN
    security_opt:
      - apparmor:unconfined
    volumes:
      - ./config.yaml:/data/config.yaml
      - ./rclone.conf:/data/rclone.conf
      - ./local-dev/mnt/rclone:/mnt/rclone:shared
      - ./local-dev/caches/rclone:/caches/rclone
    environment:
      # General
      RCLONE_BUFFER_SIZE: 0
      RCLONE_BWLIMIT: 100M
      RCLONE_BIND: 0.0.0.0
      RCLONE_LOG_LEVEL: INFO
      RCLONE_CACHE_DIR: /caches/rclone
      RCLONE_DIR_CACHE_TIME: 10s
      RCLONE_TIMEOUT: 10m
      RCLONE_UMASK: 002
      RCLONE_UID: 1000
      RCLONE_GID: 1000
      RCLONE_ALLOW_NON_EMPTY: "true"
      RCLONE_ALLOW_OTHER: "true"
      RCLONE_CONFIG: /data/rclone.conf

      # RCD API
      RCLONE_RC_ADDR: :5572
      RCLONE_RC_NO_AUTH: "true"
      RCLONE_RC_WEB_GUI: "true"
      RCLONE_RC_WEB_GUI_NO_OPEN_BROWSER: "true"

      # VFS Defaults
      RCLONE_VFS_MIN_FREE_SPACE: off
      RCLONE_VFS_CACHE_MAX_AGE: 24h
      RCLONE_VFS_MAX_CACHE_SIZE: 100G
      RCLONE_VFS_CACHE_MODE: full
      RCLONE_VFS_READ_CHUNK_LIMIT: 64M
      RCLONE_VFS_READ_CHUNK_SIZE: 5M

      # Mount Defaults
      RCLONE_NO_TRAVERSE: "true"
      RCLONE_IGNORE_EXISTING: "true"
      RCLONE_POLL_INTERVAL: 0
```

---

## Configuration
### config.yaml
```yaml
mounts:
  - backendName: "AllDebrid"
    mountPoint: "/mnt/rclone/alldebrid"

serves:
  - backendName: "AllDebrid"
    protocol: "webdav"
    addr: "0.0.0.0:8080"
```

- **`mounts`** – Specifies which Rclone remote backends to mount and where to mount them.
- **`serves`** – Configures Rclone to serve mounted directories over specified protocols.
- **Both sections are optional** – The application can run without either, or with either one of them!

---

## Environment Variables
Rclone parameters can be controlled entirely through environment variables. Below are the key variables:

### General Options
| Variable                  | Description                            | Default     |
|--------------------------|----------------------------------------|-------------|
| `RCLONE_BUFFER_SIZE`      | Buffer size for uploads/downloads      | `0`         |
| `RCLONE_BWLIMIT`          | Bandwidth limit                        | `100M`      |
| `RCLONE_BIND`             | IP to bind                             | `0.0.0.0`   |
| `RCLONE_LOG_LEVEL`        | Logging level                          | `INFO`      |
| `RCLONE_CACHE_DIR`        | Cache directory                        | `/caches/rclone` |
| `RCLONE_DIR_CACHE_TIME`   | Directory cache duration               | `10s`       |
| `RCLONE_TIMEOUT`          | Operation timeout                      | `10m`       |
| `RCLONE_UMASK`            | File permissions mask                  | `002`       |
| `RCLONE_UID`              | User ID for mount directories          | `1000`      |
| `RCLONE_GID`              | Group ID for mount directories         | `1000`      |


### RCD API Options
| Variable                  | Description                            | Default     |
|--------------------------|----------------------------------------|-------------|
| `RCLONE_RC_ADDR`          | RCD API address                        | `:5572`     |
| `RCLONE_RC_NO_AUTH`       | Disable RCD API authentication         | `true`      |
| `RCLONE_RC_WEB_GUI`       | Enable the web GUI                     | `true`      |
| `RCLONE_RC_WEB_GUI_NO_OPEN_BROWSER` | Disable browser auto-open         | `true`      |

> [!TIP]
> Did you know? All of rclone can be controlled via environmental variables.
> You can see in the compose here I have set my common VFS options as well as the mount options.
> This extends beyond pure rclone settings, to even dynamic config file entries!
> For a full list of options, see the [Rclone env-var documentation](https://rclone.org/docs/#environment-variables).


---

## Running the Application
```bash
# Edit the config.yaml file and define your mappings
$ nano config.yaml

# Start the app
$ docker compose up -d

# View logs
$ docker logs -f rclone-manager

# Stop the app
$ docker compose down
```

---

## Graceful Shutdown
- The container listens for `SIGTERM` to gracefully stop all mounted directories and serve processes before exiting.
- This prevents stale mounts and ensures clean shutdowns.

---

## Monitoring
- The application monitors the RCD process and mounts continuously.
- If RCD dies unexpectedly, it is restarted with a grace period to avoid unnecessary restarts, remounting all mounts after cleanly unmounting them.
- Serve processes are monitored separately to ensure continued operation, even if RCD restarts.
- If a serve process dies, it is restarted automatically.

---

Enjoy automated Rclone management! 🚀

