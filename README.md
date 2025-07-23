# Matrix Hostex Bridge

A Matrix bridge for the Hostex property management system using mautrix-go bridgev2 framework. This bridge allows you to receive and send messages from your Hostex conversations directly in Matrix with full **double puppeting support** - your host messages appear as sent by you, not the bridge bot.

## Features

- ✅ **Bidirectional messaging** - Send and receive messages between Matrix and Hostex
- ✅ **Real-time sync** - New messages appear in Matrix within 30 seconds
- 🔧 **Image attachments** - Images from Hostex currently show as "(empty message)" - under investigation ([#5](https://github.com/keithah/matrix-hostex-bridge/issues/5))
- ✅ **Property-prefixed rooms** - Rooms are named with property prefix: "(Property Name) - Guest Name"
- ✅ **Beeper integration** - Full compatibility with Beeper's bridge-manager
- ✅ **Message backfilling** - Historical messages are imported when creating rooms
- ✅ **Echo prevention** - Prevents duplicate messages when sending from Matrix
- ✅ **Efficient polling** - Only processes conversations with new messages
- ✅ **Manual refresh command** - Force conversation cache refresh with `!hostex refresh`
- ✅ **Double puppeting** - **WORKING!** Host messages appear as sent by you (not bridge bot) when using Beeper
- ✅ **HTTP endpoints** - Webhook support for real-time notifications from Hostex

## Architecture

```
Matrix Room ↔ mautrix-hostex ↔ Hostex API
                    ↑
              Hostex Webhooks
```

## Components

- **Hostex API Client** (`pkg/hostexapi/`): HTTP client for Hostex API v3.0.0
- **Bridge Connector** (`pkg/connector/`): mautrix bridgev2 implementation
- **Webhook Handler** (`pkg/webhook/`): Real-time event processing
- **Main Application** (`cmd/mautrix-hostex/`): Entry point and bridge initialization

## Quick Start

### Using Docker

1. Create a `docker-compose.yml`:

```yaml
version: '3.8'
services:
  mautrix-hostex:
    build: .
    container_name: mautrix-hostex
    volumes:
      - ./data:/data
    ports:
      - "29337:29337"
    restart: unless-stopped
```

2. Run the container:

```bash
docker-compose up -d
```

3. Edit the generated `data/config.yaml` with your Matrix homeserver and Hostex API details
4. Restart the container

### Manual Build

#### Prerequisites

- Go 1.21 or higher
- libolm (for Matrix encryption support)

**Install libolm on macOS:**
```bash
brew install libolm
```

**Install libolm on Ubuntu/Debian:**
```bash
sudo apt install libolm-dev
```

1. Clone and build the bridge:

```bash
git clone https://github.com/keithah/matrix-hostex-bridge.git
cd matrix-hostex-bridge

# Build with proper CGO flags for libolm
CGO_CFLAGS="-I$(brew --prefix)/include" CGO_LDFLAGS="-L$(brew --prefix)/lib" go build -o mautrix-hostex ./cmd/mautrix-hostex
```

2. Generate config (for Beeper usage):

```bash
# Use bbctl to generate proper bridgev2 config with double puppeting
bbctl config --type bridgev2 sh-generic -o config.yaml

# Or generate basic config for standalone usage
./mautrix-hostex -g -c config.yaml -r registration.yaml
```

3. Edit `config.yaml` with your settings and start:

```bash
./mautrix-hostex -c config.yaml
```

**Note**: When using with Beeper, the bbctl-generated config includes proper double puppeting configuration and should be preferred over the basic generated config.

## Configuration

### Beeper Configuration (Recommended)

When using with Beeper, the bridge uses a bridgev2 configuration that includes:

- **Double puppeting**: Automatic using `as_token` 
- **Proper bridge naming**: Uses `sh-hostex` prefix for Beeper compatibility
- **HTTP endpoints**: Webhook support at `/_matrix/mau/hostex/webhook`
- **Network section**: Hostex-specific settings

Example network section to add to your bbctl-generated config:

```yaml
network:
    hostex_api_url: https://api.hostex.io/v3
    admin_user: "@yourusername:beeper.com"
```

### Manual Configuration

For standalone usage:

- **Matrix Homeserver**: Your Matrix server URL and credentials  
- **Hostex API Token**: Your Hostex access token from the API settings

### Hostex API Setup

1. Log into your Hostex account
2. Go to API Settings
3. Generate an access token

### Setting up Bridge Avatar

The bridge uses the official Hostex logo as its avatar. To set this up:

1. Start your bridge: `./mautrix-hostex`
2. Upload the logo using the provided script: `./upload-avatar.sh`
3. Update your `config.yaml` with the returned mxc:// URI
4. Restart the bridge to apply the new avatar

## Usage

### Starting the Bridge

```bash
./mautrix-hostex
```

### Authentication

1. Start a chat with the bridge bot (e.g., `@sh-hostexbot:beeper.local`)
2. Use the `login` command and provide your Hostex API token
3. The bridge will automatically create rooms for your active conversations

### Available Commands

- `login` - Authenticate with your Hostex API token
- `logout` - Sign out from Hostex
- `list-logins` - Show your current login status
- `refresh` - Manually refresh conversation cache and check for new messages
- `help` - Show available commands

### Sending Messages

- **Text messages** - Simply type in any bridged room
- **Images** - Send images directly in Matrix (they'll appear in Hostex)
- **Mixed content** - Send text with images attached

## Using with Beeper

This bridge is fully compatible with Beeper and includes proper double puppeting support:

### Recommended Setup (bbctl)

1. Install bbctl and login: `bbctl login`
2. Generate proper bridgev2 config: `bbctl config --type bridgev2 sh-generic -o config.yaml`
3. Update the generated config:
   - Change `username_template` to `sh-hostex_{{.}}`
   - Change bot `username` to `sh-hostexbot`
   - Add the `network` section with your Hostex API settings
4. Register and start your bridge
5. The bridge connects via websocket to Beeper's infrastructure with full double puppeting

### Double Puppeting

When properly configured with the bridgev2 config:
- **Host messages** appear as sent by your actual Matrix user (not the bridge bot)
- **Guest messages** appear as received from the guest
- Uses Beeper's `as_token` for automatic double puppeting
- No additional client setup required

### Manual Setup

1. Register the bridge: `bbctl register sh-hostex`
2. Configure your bridge locally with the provided config template
3. Ensure double puppeting is enabled in the config

## Development

### Project Structure

```
hostex-matrix-bridge/
├── cmd/mautrix-hostex/main.go     # Application entry point
├── pkg/
│   ├── connector/
│   │   ├── connector.go           # Main bridge implementation with double puppeting
│   │   └── minimal.go             # Minimal test connector
│   ├── hostexapi/client.go        # Hostex API client
│   └── webhook/handler.go         # Webhook processing
├── config.yaml                    # Bridgev2 configuration (generated by bbctl)
├── config.example.yaml           # Example standalone configuration
├── registration.yaml              # Bridge registration for Matrix homeserver
├── go.mod                         # Go dependencies
├── Dockerfile                     # Container build
└── docker-run.sh                  # Container runtime
```

### Building

```bash
# Development build
go run . -g -c config.yaml -r registration.yaml

# Production build with version info
TAG=v0.1.0 COMMIT=$(git rev-parse HEAD) BUILD_TIME=$(date -u +%Y-%m-%dT%H:%M:%SZ) \
go build -ldflags "-X 'hostex-matrix-bridge/cmd/mautrix-hostex.Tag=${TAG}' -X 'hostex-matrix-bridge/cmd/mautrix-hostex.Commit=${COMMIT}' -X 'hostex-matrix-bridge/cmd/mautrix-hostex.BuildTime=${BUILD_TIME}'"
```

### Testing

```bash
# Run tests
go test ./...

# Test Hostex API connection
curl -H "Hostex-Access-Token: your-token" https://api.hostex.io/v1/properties
```

## License

This project is licensed under the Mozilla Public License 2.0 (MPL-2.0) - see the LICENSE file for details.

This license choice matches the core mautrix framework dependency and allows commercial use while ensuring improvements to the bridge remain open source.

## Support

- [Matrix Room]: #hostex-bridge:your-homeserver.com
- [Issues]: https://github.com/keithah/matrix-hostex-bridge/issues
- [Hostex API Docs]: https://hostex-openapi.readme.io/
- [Troubleshooting Guide]: ./TROUBLESHOOTING.md
- [Changelog]: ./CHANGELOG.md

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request