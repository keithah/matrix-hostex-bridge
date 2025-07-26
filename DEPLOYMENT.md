# 🚀 Hostex Bridge Docker Compose Deployment

## 📂 Files to Copy

Copy these files to your Docker Compose setup:

```bash
# Main deployment files
docker-compose.yml          # Main Docker Compose configuration
deployment/README.md         # Detailed setup guide
deployment/config.example.yaml  # Configuration template

# Helper scripts
deployment/scripts/setup.sh     # Automated setup script
deployment/scripts/backup.sh    # Backup script
deployment/.env.example         # Environment variables template
```

## 🎯 Quick Deployment Commands

### 1. Copy Files to Your Server

```bash
# Create deployment directory
mkdir -p ~/docker/hostex-bridge
cd ~/docker/hostex-bridge

# Copy all deployment files from this repository
scp -r user@build-server:/path/to/hostex-bridge-dev/docker-compose.yml .
scp -r user@build-server:/path/to/hostex-bridge-dev/deployment .

# Or clone the repository
git clone https://github.com/keithah/matrix-hostex-bridge.git temp
cp temp/docker-compose.yml .
cp -r temp/deployment .
rm -rf temp
```

### 2. Automated Setup (Recommended)

```bash
# Make setup script executable
chmod +x deployment/scripts/setup.sh

# Run the interactive setup
./deployment/scripts/setup.sh
```

### 3. Manual Setup (Alternative)

```bash
# Create directories
mkdir -p data logs

# Copy environment template
cp deployment/.env.example .env

# Edit environment variables
nano .env

# Generate initial config
docker-compose up -d
# Wait for "Generated config files" message, then stop
docker-compose stop

# Edit configuration
cp deployment/config.example.yaml data/config.yaml
nano data/config.yaml

# Start the bridge
docker-compose up -d
```

## ⚙️ Configuration Steps

### 1. Edit Configuration

Update `data/config.yaml` with your settings:

```yaml
# Your Matrix homeserver
homeserver:
    address: https://matrix.your-domain.com
    domain: your-domain.com

# Bridge network settings  
network:
    hostex_api_url: https://api.hostex.io/v3
    admin_user: "@your-username:your-domain.com"

# Database (SQLite is fine for most setups)
database:
    type: sqlite3-fk-wal
    uri: file:mautrix-hostex.db?_txlock=immediate
```

### 2. Matrix Server Setup

Copy the registration file to your Matrix server:

```bash
# Copy registration to your Matrix server
scp data/registration.yaml matrix-server:/path/to/synapse/

# Add to homeserver.yaml
app_service_config_files:
  - /path/to/registration.yaml

# Restart Matrix server
systemctl restart matrix-synapse
```

### 3. Start and Login

```bash
# Start the bridge
docker-compose up -d

# Check logs
docker-compose logs -f hostex-bridge

# In Matrix, find @sh-hostexbot:your-domain.com
# Send: login
# Enter your Hostex API token when prompted
```

## 🔧 Management Commands

```bash
# View logs
docker-compose logs -f hostex-bridge

# Restart bridge
docker-compose restart hostex-bridge

# Update to latest version
docker-compose pull && docker-compose up -d

# Check health
curl http://localhost:29337/_matrix/mau/hostex/health

# Create backup
./deployment/scripts/backup.sh

# Stop bridge
docker-compose stop hostex-bridge

# Remove everything (careful!)
docker-compose down -v
```

## 🐳 Docker Compose Configuration

The `docker-compose.yml` includes:

- **Health checks** - Automatic restart if bridge fails
- **Volume mounts** - Persistent data and logs
- **Resource limits** - Prevents resource exhaustion
- **Proper networking** - Isolated bridge network
- **Logging configuration** - Manageable log rotation

Optional PostgreSQL database is included but commented out.

## 📊 Monitoring Setup (Optional)

Add monitoring to your `docker-compose.yml`:

```yaml
services:
  hostex-bridge:
    environment:
      - METRICS_ENABLED=true
    ports:
      - "8080:8080"  # Metrics endpoint
      
  # Add Prometheus/Grafana for monitoring
  prometheus:
    image: prom/prometheus
    ports:
      - "9090:9090"
    volumes:
      - ./prometheus.yml:/etc/prometheus/prometheus.yml
```

## 🔒 Security Considerations

1. **Firewall**: Only expose bridge port to Matrix server
2. **API Keys**: Store Hostex API token securely
3. **Network**: Use Docker networks to isolate services
4. **Updates**: Keep bridge image updated regularly
5. **Backups**: Regular backups of configuration and database

## 🐛 Troubleshooting

### Common Issues:

**Bridge won't start:**
```bash
# Check logs for errors
docker-compose logs hostex-bridge

# Verify configuration syntax
docker-compose config
```

**Matrix server can't reach bridge:**
```bash
# Check network connectivity
docker-compose exec hostex-bridge ping matrix-server

# Verify port is accessible
curl http://your-server:29337/_matrix/mau/hostex/health
```

**Authentication fails:**
```bash
# Verify API token is correct
# Check Hostex API permissions
# Look for rate limiting in logs
```

### Log Locations:
- Container logs: `docker-compose logs hostex-bridge`
- File logs: `./logs/bridge.log`
- Matrix server logs: Check your Matrix server documentation

## 📈 Production Recommendations

1. **Use PostgreSQL** for better performance with multiple users
2. **Set up log rotation** to prevent disk space issues
3. **Monitor resource usage** with metrics
4. **Implement automated backups** using the provided script
5. **Use reverse proxy** (nginx/Traefik) for SSL termination
6. **Set up alerting** for bridge failures

## 🔄 Updates and Maintenance

The bridge automatically uses the `:latest` tag. To update:

```bash
# Pull latest image
docker-compose pull hostex-bridge

# Recreate container with new image  
docker-compose up -d hostex-bridge

# Verify it's working
docker-compose logs -f hostex-bridge
```

For automatic updates, consider using Watchtower or similar tools.

---

## 📞 Need Help?

- **Documentation**: See `deployment/README.md` for detailed setup
- **Issues**: [GitHub Issues](https://github.com/keithah/matrix-hostex-bridge/issues)
- **Matrix Support**: Join bridge development rooms

Your Hostex bridge should now be running and ready to connect your Hostex conversations to Matrix! 🎉