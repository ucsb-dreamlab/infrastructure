#!/bin/env bash

# Logging function
log() {
    echo "$(date '+%Y-%m-%d %H:%M:%S') - $1"
}


# =============================================================================
# PRE-CLEANUP TASKS
# =============================================================================

log "Pre-cleanup Tasks"

# Stop unnecessary services before cleanup
log "Stopping services for cleanup..."
systemctl stop rsyslog || true
systemctl stop cron || true

# Remove temporary files created during installation
log "Removing temporary installation files..."
rm -rf /tmp/*
rm -rf /var/tmp/*

# =============================================================================
# DISK CLEANUP AND IMAGE PREPARATION
# =============================================================================

log "Disk Cleanup and Image Preparation"

# Clean package cache
log "Cleaning package cache..."
apt-get clean
apt-get autoclean
apt-get autoremove -y

# Remove old kernels (keep current and one backup)
log "Removing old kernels..."
apt-get autoremove --purge -y

# Clear logs
log "Clearing system logs..."
journalctl --rotate
journalctl --vacuum-time=1s
find /var/log -type f -name "*.log" -exec truncate -s 0 {} \;
find /var/log -type f -name "*.log.*" -delete
rm -rf /var/log/apt/*
rm -rf /var/log/unattended-upgrades/*

# Clear bash history
log "Clearing bash history..."
history -c
history -w
rm -f /root/.bash_history
find /home -name ".bash_history" -delete 2>/dev/null || true

# Clear SSH host keys (will be regenerated on first boot)
log "Removing SSH host keys..."
rm -f /etc/ssh/ssh_host_*

# Clear machine-id (will be regenerated)
log "Clearing machine-id..."
truncate -s 0 /etc/machine-id
rm -f /var/lib/dbus/machine-id

# Clear network configuration
log "Clearing network interface configurations..."
rm -f /etc/udev/rules.d/70-persistent-net.rules
rm -f /etc/netplan/*.yaml.bak 2>/dev/null || true

# Remove cloud-init artifacts (if present)
if [ -d /var/lib/cloud ]; then
    log "Cleaning cloud-init artifacts..."
    cloud-init clean --logs || true
    rm -rf /var/lib/cloud/instances/*
    rm -rf /var/lib/cloud/data/*
fi

# Clear temporary files and caches
log "Clearing temporary files and caches..."
rm -rf /tmp/*
rm -rf /var/tmp/*
rm -rf /root/.cache
find /home -name ".cache" -type d -exec rm -rf {} \; 2>/dev/null || true

# Clear package lists
log "Clearing package lists..."
rm -rf /var/lib/apt/lists/*

# Clear systemd journal
log "Clearing systemd journal..."
journalctl --flush --rotate
journalctl --vacuum-time=1s

# Zero out free space for better compression
log "Zeroing out free space (this may take a while)..."
dd if=/dev/zero of=/EMPTY bs=1M 2>/dev/null || true
rm -f /EMPTY

# Clear swap if present
if [ -f /swapfile ]; then
    log "Clearing swap file..."
    swapoff /swapfile
    dd if=/dev/zero of=/swapfile bs=1M count=1 2>/dev/null || true
    mkswap /swapfile
fi

# Final cleanup
log "Performing final cleanup..."
sync

# Clear the preparation log itself (optional)
# rm -f "$LOG_FILE"

log "VM preparation completed successfully!"
log "The VM is now ready for export as a base image."

echo "==============================================="
echo "VM PREPARATION COMPLETE!"
echo "==============================================="
echo "Next steps:"
echo "1. Shutdown the VM: sudo shutdown -h now"
echo "2. Export/clone the VM disk image"
echo "3. The base image is ready for deployment"
echo "==============================================="

# Optional: Auto-shutdown after completion
# shutdown -h +1 "VM preparation complete. Shutting down in 1 minute."