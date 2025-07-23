# Matrix Hostex Bridge

A Matrix bridge for the Hostex property management system using mautrix-go bridgev2 framework. This bridge allows you to receive and send messages from your Hostex conversations directly in Matrix.

## Features

- ✅ **Bidirectional messaging** - Send and receive messages between Matrix and Hostex
- ✅ **Real-time sync** - New messages appear in Matrix within 30 seconds
- ✅ **Image attachments** - Images from Hostex display properly in Matrix (recently fixed!)
- ✅ **Property-prefixed rooms** - Rooms are named with property prefix: "(Property Name) - Guest Name"
- ✅ **Beeper integration** - Full compatibility with Beeper's bridge-manager
- ✅ **Message backfilling** - Historical messages are imported when creating rooms
- ✅ **Echo prevention** - Prevents duplicate messages when sending from Matrix
- ✅ **Efficient polling** - Only processes conversations with new messages
- ✅ **Manual refresh command** - Force conversation cache refresh with `!hostex refresh`

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

2. Generate config:

```bash
./mautrix-hostex -g -c config.yaml -r registration.yaml
```

3. Edit `config.yaml` and start:

```bash
./mautrix-hostex -c config.yaml
```

## Configuration

### Required Settings

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

This bridge is compatible with Beeper's bridge-manager:

1. Install bbctl and login: `bbctl login`
2. Register the bridge: `bbctl register sh-hostex`
3. Configure your bridge locally
4. The bridge connects via websocket to Beeper's infrastructure

## Development

### Project Structure

```
hostex-matrix-bridge/
├── cmd/mautrix-hostex/main.go     # Application entry point
├── pkg/
│   ├── connector/connector.go     # Bridge implementation
│   ├── hostexapi/client.go        # Hostex API client
│   └── webhook/handler.go         # Webhook processing
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
- [Issues]: https://github.com/your-org/hostex-matrix-bridge/issues
- [Hostex API Docs]: https://hostex-openapi.readme.io/

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request