# MyFeed - RSS Reader

A zero-maintenance RSS feed reader inspired by Feedly, designed for self-hosted deployment on DigitalOcean.

## Current Status

🚧 **Under Development** - This is currently a placeholder deployment to establish the deployment pipeline.

## Quick Start

```bash
# Run locally
go run main.go

# Visit http://localhost:8080
```

## Deployment

This application is configured for deployment on DigitalOcean App Platform with automatic builds from the GitHub repository.

## Architecture

- **Backend**: Go with built-in HTTP server
- **Frontend**: Vanilla HTML/CSS/JS (will migrate to React)
- **Database**: Will use SQLite
- **Deployment**: DigitalOcean App Platform

## Security

This is a **public repository**. Security measures in place:

- Comprehensive `.gitignore` to prevent accidental commit of sensitive files
- No hardcoded secrets, API keys, or credentials in source code
- Environment variables used for configuration
- Database files and data directories excluded from git

⚠️ **Never commit:**
- API keys or access tokens
- Database credentials
- SSL certificates or private keys
- Environment files (.env, config files with secrets)
- User data or database files

## Roadmap

See [PROJECT_PLAN.md](PROJECT_PLAN.md) for detailed feature planning and implementation phases.

## API Endpoints

### Current (Placeholder)
- `GET /` - Frontend application
- `GET /api/health` - Health check
- `GET /api/feeds` - Placeholder feeds endpoint

### Planned
- Full RSS feed management
- Article reading and organization
- Search functionality
- OPML import/export