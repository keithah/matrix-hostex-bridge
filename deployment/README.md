# Hostex Bridge Docker Compose Deployment

This guide walks you through deploying the Hostex Matrix bridge using Docker Compose.

## 📋 Prerequisites

- Docker and Docker Compose installed
- Hostex API access token
- Matrix homeserver with application service support (Synapse, Dendrite, etc.)

## 🚀 Quick Start

### 1. Copy Files to Your Server

Copy these files to your deployment directory:

```bash
# Create deployment directory
mkdir -p ~/docker/hostex-bridge
cd ~/docker/hostex-bridge

# Copy the deployment files
cp docker-compose.yml .
cp -r deployment/* .
```

### 2. Initial Setup

```bash
# Create data directory
mkdir -p data logs

# Start the container for initial config generation
docker-compose up -d

# The container will exit after generating config files
# Check the logs to see the message
docker-compose logs
```

### 3. Configure the Bridge

Edit the generated configuration files:

```bash
# Edit bridge configuration
nano data/config.yaml

# Edit registration file (copy this to your Matrix server)
nano data/registration.yaml
```

### 4. Matrix Server Setup

Copy the registration file to your Matrix server and add it to your homeserver configuration:

**For Synapse** (`homeserver.yaml`):
```yaml
app_service_config_files:
  - /path/to/registration.yaml
```

**For Dendrite** (`dendrite.yaml`):
```yaml
app_service_api:
  config_files:
    - /path/to/registration.yaml
```

### 5. Start the Bridge

```bash
# Start the bridge
docker-compose up -d

# Check logs
docker-compose logs -f hostex-bridge
```

### 6. Login to the Bridge

1. Find the bridge bot in Matrix (usually `@sh-hostexbot:your-domain.com`)
2. Send `login` to start the authentication process
3. Enter your Hostex API token when prompted

## 📁 Directory Structure

After deployment, your directory should look like:

```
~/docker/hostex-bridge/
├── docker-compose.yml
├── data/
│   ├── config.yaml
│   ├── registration.yaml
│   └── mautrix-hostex.db
├── logs/
│   └── bridge.log
└── deployment/
    ├── README.md
    ├── config.example.yaml
    └── scripts/
```

## ⚙️ Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `UID` | `1000` | User ID for file permissions |
| `GID` | `1000` | Group ID for file permissions |
| `BRIDGEV2` | `1` | Enable bridgev2 mode |

### Important Configuration Options

Edit `data/config.yaml`:

```yaml
# Homeserver configuration
homeserver:
    address: https://matrix.your-domain.com
    domain: your-domain.com

# Database (default: SQLite)
database:
    type: sqlite3
    uri: mautrix-hostex.db
    
# Or use PostgreSQL:
# database:
#     type: postgres
#     uri: postgres://mautrix:password@postgres:5432/mautrix_hostex

# Bridge configuration
bridge:
    username_template: "sh-hostex_{{.}}"
    displayname_template: "{{.DisplayName}} (Hostex)"
    
# Network-specific configuration
network:
    hostex_api_url: https://api.hostex.io/v3
    # Add other Hostex-specific settings here
```

## 🔧 Management Commands

### View Logs
```bash
docker-compose logs -f hostex-bridge
```

### Restart Bridge
```bash
docker-compose restart hostex-bridge
```

### Update Bridge
```bash
docker-compose pull
docker-compose up -d
```

### Backup Data
```bash
tar -czf hostex-bridge-backup-$(date +%Y%m%d).tar.gz data/
```

### Health Check
```bash
curl http://localhost:29337/_matrix/mau/hostex/health
```

## 🐛 Troubleshooting

### Bridge Won't Start

1. **Check logs**: `docker-compose logs hostex-bridge`
2. **Verify config**: Ensure `config.yaml` is properly formatted
3. **Check permissions**: Ensure data directory is writable

### Connection Issues

1. **Port conflicts**: Change the port mapping in `docker-compose.yml`
2. **Network issues**: Ensure Matrix server can reach the bridge
3. **Registration**: Verify `registration.yaml` is loaded by Matrix server

### Authentication Issues

1. **API Token**: Verify your Hostex API token is correct
2. **Permissions**: Ensure token has necessary permissions
3. **Rate Limits**: Check if you're hitting API rate limits

### Common Log Messages

- `"Generated config files"` - Normal, edit config and restart
- `"Bridge started"` - Successfully running
- `"Failed to authenticate"` - Check API token
- `"Connecting to Hostex"` - Normal startup sequence

## 🔒 Security Considerations

1. **Firewall**: Only expose port 29337 to your Matrix server
2. **API Keys**: Store API tokens securely, never in public repositories
3. **Database**: Use strong passwords for PostgreSQL if enabled
4. **Updates**: Keep the bridge image updated regularly

## 📊 Monitoring

### Prometheus Metrics (Optional)

Add to `docker-compose.yml`:

```yaml
services:
  hostex-bridge:
    environment:
      - METRICS_ENABLED=true
      - METRICS_LISTEN=0.0.0.0:8080
    ports:
      - "8080:8080"  # Metrics port
```

### Log Aggregation

The bridge logs to both console and file. Configure your log aggregation system to collect from:
- Docker logs: `docker-compose logs`
- File logs: `./logs/bridge.log`

## 🔄 Updates and Maintenance

### Automatic Updates (Optional)

Use Watchtower for automatic updates:

```yaml
services:
  watchtower:
    image: containrrr/watchtower
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - WATCHTOWER_POLL_INTERVAL=3600
      - WATCHTOWER_INCLUDE_STOPPED=true
    restart: unless-stopped
```

### Manual Updates

```bash
# Pull latest image
docker-compose pull hostex-bridge

# Recreate container with new image
docker-compose up -d hostex-bridge
```

## 📞 Support

- **Issues**: [GitHub Issues](https://github.com/keithah/matrix-hostex-bridge/issues)
- **Documentation**: Check the project README
- **Matrix Community**: Join the bridge development rooms