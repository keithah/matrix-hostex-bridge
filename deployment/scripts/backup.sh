#!/bin/bash

# Hostex Bridge Backup Script
# Creates timestamped backups of bridge data

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Configuration
BACKUP_DIR="./backups"
TIMESTAMP=$(date +"%Y%m%d_%H%M%S")
BACKUP_NAME="hostex-bridge-backup-${TIMESTAMP}"

# Create backup directory
mkdir -p "$BACKUP_DIR"

log_info "Starting backup of Hostex bridge data..."

# Stop the bridge to ensure consistent backup
log_info "Stopping bridge for consistent backup..."
docker-compose stop hostex-bridge

# Create the backup
log_info "Creating backup archive..."
tar -czf "${BACKUP_DIR}/${BACKUP_NAME}.tar.gz" \
    data/ \
    logs/ \
    docker-compose.yml \
    .env 2>/dev/null || true

# Restart the bridge
log_info "Restarting bridge..."
docker-compose start hostex-bridge

# Verify backup
if [ -f "${BACKUP_DIR}/${BACKUP_NAME}.tar.gz" ]; then
    BACKUP_SIZE=$(du -h "${BACKUP_DIR}/${BACKUP_NAME}.tar.gz" | cut -f1)
    log_success "Backup created: ${BACKUP_NAME}.tar.gz (${BACKUP_SIZE})"
    
    # Clean up old backups (keep last 5)
    log_info "Cleaning up old backups (keeping last 5)..."
    cd "$BACKUP_DIR"
    ls -t hostex-bridge-backup-*.tar.gz | tail -n +6 | xargs rm -f 2>/dev/null || true
    cd ..
    
    log_success "Backup complete!"
    echo "Backup location: ${BACKUP_DIR}/${BACKUP_NAME}.tar.gz"
else
    log_error "Backup failed!"
    exit 1
fi