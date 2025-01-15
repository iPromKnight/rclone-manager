#!/bin/sh

PUID=${PUID:-1000}
PGID=${PGID:-1000}

if ! getent group customgroup >/dev/null 2>&1; then
    addgroup -g "$PGID" rclonemanager
fi

if ! id -u rclonemanager >/dev/null 2>&1; then
    adduser -u "$PUID" -G rclonemanager -D -s /bin/sh rclonemanager
fi

chown -R rclonemanager:rclonemanager /data

exec su-exec rclonemanager "/usr/local/bin/rclone-manager"
