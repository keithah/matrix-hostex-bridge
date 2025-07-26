#!/bin/bash

# Hostex Bridge Docker Compose Setup Script
# This script helps you set up the Hostex Matrix bridge

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Banner
echo -e "${BLUE}"
echo "╔══════════════════════════════════════╗"
echo "║      Hostex Bridge Setup Script     ║"
echo "║        Docker Compose Version       ║"
echo "╚══════════════════════════════════════╝"
echo -e "${NC}"

# Check prerequisites
log_info "Checking prerequisites..."

if ! command -v docker &> /dev/null; then
    log_error "Docker is not installed. Please install Docker first."
    exit 1
fi

if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    log_error "Docker Compose is not installed. Please install Docker Compose first."
    exit 1
fi

log_success "Prerequisites check passed"

# Create directory structure
log_info "Creating directory structure..."
mkdir -p data logs deployment/scripts
log_success "Directory structure created"

# Check if config already exists
if [ -f "data/config.yaml" ]; then
    log_warning "Configuration file already exists at data/config.yaml"
    read -p "Do you want to overwrite it? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Keeping existing configuration"
        SKIP_CONFIG=true
    fi
fi

# Generate initial config if needed
if [ "$SKIP_CONFIG" != "true" ]; then
    log_info "Starting container to generate initial configuration..."
    
    # Pull the latest image
    docker-compose pull hostex-bridge
    
    # Start container briefly to generate config
    docker-compose up --no-deps hostex-bridge &
    COMPOSE_PID=$!
    
    # Wait for config generation
    sleep 10
    
    # Stop the container
    docker-compose stop hostex-bridge
    wait $COMPOSE_PID 2>/dev/null || true
    
    log_success "Initial configuration generated"
fi

# Check if config was generated
if [ ! -f "data/config.yaml" ]; then
    log_error "Configuration file was not generated. Please check the logs."
    docker-compose logs hostex-bridge
    exit 1
fi

# Interactive configuration
log_info "Let's configure your bridge..."

echo
echo -e "${YELLOW}Matrix Homeserver Configuration${NC}"
read -p "Enter your Matrix homeserver URL (e.g., https://matrix.example.com): " HOMESERVER_URL
read -p "Enter your Matrix domain (e.g., example.com): " MATRIX_DOMAIN

echo
echo -e "${YELLOW}Bridge Configuration${NC}"
read -p "Enter your admin Matrix user ID (e.g., @admin:example.com): " ADMIN_USER

echo
echo -e "${YELLOW}Database Configuration${NC}"
echo "1) SQLite (recommended)"
echo "2) PostgreSQL"
read -p "Choose database type (1-2): " DB_CHOICE

if [ "$DB_CHOICE" = "2" ]; then
    log_info "PostgreSQL selected. Uncomment the postgres service in docker-compose.yml"
    read -p "Enter PostgreSQL password: " -s POSTGRES_PASSWORD
    echo
fi

# Update configuration
log_info "Updating configuration file..."

# Use sed to update the config file
sed -i.bak \
    -e "s|address: https://matrix.example.com|address: $HOMESERVER_URL|g" \
    -e "s|domain: example.com|domain: $MATRIX_DOMAIN|g" \
    -e "s|admin_user: \"@admin:example.com\"|admin_user: \"$ADMIN_USER\"|g" \
    data/config.yaml

if [ "$DB_CHOICE" = "2" ]; then
    sed -i.bak2 \
        -e "s|type: sqlite3-fk-wal|type: postgres|g" \
        -e "s|uri: file:mautrix-hostex.db?_txlock=immediate|uri: postgres://mautrix:$POSTGRES_PASSWORD@postgres:5432/mautrix_hostex|g" \
        data/config.yaml
fi

log_success "Configuration updated"

# Start the bridge
log_info "Starting the Hostex bridge..."
docker-compose up -d

# Wait a moment for startup
sleep 5

# Check if it's running
if docker-compose ps hostex-bridge | grep -q "Up"; then
    log_success "Bridge is running!"
    
    echo
    echo -e "${GREEN}╔═══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║                        SETUP COMPLETE!                       ║${NC}"
    echo -e "${GREEN}╚═══════════════════════════════════════════════════════════════╝${NC}"
    echo
    
    log_info "Next steps:"
    echo "1. Copy data/registration.yaml to your Matrix server"
    echo "2. Add the registration file to your homeserver config"
    echo "3. Restart your Matrix server"
    echo "4. Find the bridge bot (@sh-hostexbot:$MATRIX_DOMAIN) and send 'login'"
    echo "5. Enter your Hostex API token when prompted"
    
    echo
    log_info "Useful commands:"
    echo "• View logs: docker-compose logs -f hostex-bridge"
    echo "• Restart bridge: docker-compose restart hostex-bridge"
    echo "• Health check: curl http://localhost:29337/_matrix/mau/hostex/health"
    
else
    log_error "Bridge failed to start. Check the logs:"
    docker-compose logs hostex-bridge
    exit 1
fi

log_info "Setup complete! Check the logs with: docker-compose logs -f"