# Automated Release Process

This repository uses GitHub Actions to automatically build and push Docker images to Docker Hub whenever a new release is created.

## Setup Instructions

### 1. Configure Docker Hub Secrets

To enable automated Docker builds, you need to add your Docker Hub credentials as GitHub repository secrets:

1. **Navigate to Repository Settings**:
   - Go to your GitHub repository
   - Click on "Settings" tab
   - Click on "Secrets and variables" → "Actions"

2. **Add Required Secrets**:
   - `DOCKER_USERNAME`: Your Docker Hub username (`keithah`)
   - `DOCKER_PASSWORD`: Your Docker Hub personal access token

### 2. Create Docker Hub Personal Access Token

1. Log in to [Docker Hub](https://hub.docker.com/)
2. Go to Account Settings → Security
3. Click "New Access Token"
4. Name: `github-actions-hostex-bridge`
5. Permissions: `Read, Write, Delete`
6. Copy the generated token

### 3. Add Secrets to GitHub

Run these commands or use the GitHub web interface:

```bash
# Set Docker Hub username
gh secret set DOCKER_USERNAME --body "keithah"

# Set Docker Hub password (paste your access token when prompted)
gh secret set DOCKER_PASSWORD
```

## Automated Workflows

### Docker Build Workflow (`.github/workflows/docker-build.yml`)

**Triggers:**
- ✅ **Release Published**: Automatically builds when you create a GitHub release
- ✅ **Manual Dispatch**: Can be triggered manually with custom tags

**Features:**
- 🏗️ **Multi-architecture builds**: linux/amd64, linux/arm64
- 🏷️ **Smart tagging**: Semantic versions, latest tag, branch names
- 📝 **Docker Hub description updates**: Syncs README.md to Docker Hub
- ⚡ **Build caching**: Faster subsequent builds using GitHub Actions cache
- 🔧 **Build args**: Includes version, commit hash, and build timestamp

**Output:**
- `keithah/mautrix-hostex:v1.2.3` (version tag)
- `keithah/mautrix-hostex:v1.2` (minor version)
- `keithah/mautrix-hostex:v1` (major version)
- `keithah/mautrix-hostex:latest` (latest release)

### CI Workflow (`.github/workflows/ci.yml`)

**Triggers:**
- 📤 **Push to main/develop**: Runs tests on code changes
- 🔄 **Pull Requests**: Validates changes before merge

**Jobs:**
- 🧪 **Test**: Runs Go tests with proper dependencies
- 🔍 **Lint**: Code quality checks with golangci-lint
- 🐳 **Docker Build Test**: Validates Docker builds on PRs (no push)

## Creating a Release

### Method 1: GitHub Web Interface
1. Go to your repository on GitHub
2. Click "Releases" → "Create a new release"
3. Choose a tag (e.g., `v0.1.3`)
4. Write release notes
5. Click "Publish release"
6. 🚀 **GitHub Actions automatically builds and pushes Docker images**

### Method 2: GitHub CLI
```bash
# Create a new release
gh release create v0.1.3 \
  --title "v0.1.3 - Bug Fixes" \
  --notes "Fixed critical attachment processing bug"

# Docker build starts automatically
```

### Method 3: Git Tags
```bash
# Create and push a tag
git tag -a v0.1.3 -m "Release v0.1.3"
git push origin v0.1.3

# Then create release from tag on GitHub
gh release create v0.1.3 --title "v0.1.3" --generate-notes
```

## Manual Docker Build

You can also trigger a manual build without creating a release:

1. Go to Actions tab in GitHub
2. Select "Build and Push Docker Image"
3. Click "Run workflow"
4. Enter a custom tag name
5. Click "Run workflow"

## Monitoring Builds

- **GitHub Actions**: Monitor build progress in the "Actions" tab
- **Docker Hub**: Check image availability at https://hub.docker.com/r/keithah/mautrix-hostex
- **Build Status**: Workflow badges can be added to README.md

## Troubleshooting

### Common Issues:

1. **Docker Hub Login Failed**
   - Verify `DOCKER_USERNAME` and `DOCKER_PASSWORD` secrets
   - Ensure Docker Hub token has correct permissions

2. **Multi-arch Build Errors**
   - Check that base images support both amd64 and arm64
   - Verify CGO dependencies are available for both architectures

3. **Build Dependencies Missing**
   - Ensure Dockerfile includes all required system packages
   - Update package names if base image changes

### Build Logs:
- Click on failed workflow runs in GitHub Actions
- Check individual job logs for detailed error messages
- Docker build logs show compilation and dependency issues

## Security Notes

- 🔐 **Access tokens**: Use personal access tokens, not passwords
- 🔒 **Limited scope**: Docker Hub tokens should have minimal required permissions
- 🔄 **Token rotation**: Regularly rotate Docker Hub access tokens
- 📋 **Secret management**: Never commit secrets to repository code

## Next Steps

After setup, your release process becomes:
1. **Develop** → Push changes to main
2. **Test** → CI validates changes automatically  
3. **Release** → Create GitHub release
4. **Deploy** → Docker images available within minutes

The entire process from release creation to Docker Hub availability typically takes 3-5 minutes! 🚀