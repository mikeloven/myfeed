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