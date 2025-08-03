# MyFeed - Feedly Clone Project Plan

## 📋 Project Overview

MyFeed is a self-hosted RSS feed reader inspired by Feedly, designed for **zero maintenance** deployment on Digital Ocean. The focus is on delivering core RSS functionality with a clean, simple UI while avoiding complex features that require ongoing maintenance.

## 🎯 Project Goals

- **Zero Maintenance**: Deploy once, run indefinitely with minimal intervention
- **Core Functionality**: 90% of Feedly's value with 10% of the complexity
- **Self-Hosted**: Complete control over data and hosting
- **Simple UI**: Clean, functional interface optimized for reading
- **Digital Ocean Ready**: Optimized for VPS deployment

## 🛠️ Recommended Tech Stack

### **Primary Stack: Go + SQLite + React**

**Backend:**
- **Language**: Go 1.21+
- **Framework**: Gin or Echo (lightweight HTTP framework)
- **RSS Parsing**: `gofeed` library
- **Database**: SQLite with `github.com/mattn/go-sqlite3`
- **Background Jobs**: Built-in goroutines with `gocron`
- **Authentication**: JWT tokens

**Frontend:**
- **Framework**: React 18 with TypeScript
- **Build Tool**: Vite
- **State Management**: `@tanstack/react-query` for server state
- **UI Components**: `shadcn/ui` or `@headlessui/react`
- **Styling**: Tailwind CSS
- **PWA**: Workbox for offline capabilities

**Deployment:**
- **Container**: Docker multi-stage build
- **Reverse Proxy**: Nginx
- **SSL**: Let's Encrypt (certbot)
- **Monitoring**: Simple health checks

## 📦 Deployment Architecture

```yaml
# docker-compose.yml structure
services:
  myfeed:
    build: .
    ports:
      - "8080:8080"
    volumes:
      - ./data:/app/data
    restart: unless-stopped
  
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf
      - ./ssl:/etc/letsencrypt
```

**Infrastructure Requirements:**
- **Server**: Digital Ocean Droplet (2GB RAM, $12/month)
- **Storage**: 20GB SSD (included) + optional block storage for backups
- **Domain**: Personal domain pointing to droplet IP
- **Cost**: ~$13-14/month total

## ✅ Phase 1 Features (MVP)

### **Core RSS Functionality**
- ✅ RSS/Atom feed subscription
- ✅ Automatic feed discovery from URLs
- ✅ Feed health monitoring and error handling
- ✅ OPML import/export
- ✅ Feed metadata management (title, description, favicon)

### **Organization & Management**
- ✅ Unlimited folders/categories
- ✅ Drag-and-drop feed organization
- ✅ Feed and folder renaming
- ✅ Bulk operations (mark all as read, delete, move)

### **Reading Experience**
- ✅ Multiple view modes:
  - List view (compact headlines)
  - Card view (with previews)
  - Magazine view (visual layout)
- ✅ Mark as read/unread (individual and bulk)
- ✅ Save for later functionality
- ✅ Full-text search across all articles
- ✅ Date-based filtering
- ✅ Keyboard navigation (J/K keys, spacebar)
- ✅ Article content extraction and cleaning

### **User Interface**
- ✅ Responsive design (mobile-first)
- ✅ Dark/light theme toggle
- ✅ Progressive Web App (PWA) capabilities
- ✅ Keyboard shortcuts
- ✅ Clean, minimal design
- ✅ Fast loading and smooth interactions

### **Data Management**
- ✅ Article cleanup (auto-delete after 30 days)
- ✅ Database optimization routines
- ✅ Export functionality (JSON, OPML)
- ✅ Basic analytics (read counts, popular feeds)

## 🔄 Phase 2 Features (Enhancement)

### **Content Features**
- 📝 Article notes and highlights
- 🏷️ Basic tagging system
- 📊 Reading statistics and trends
- 🔗 Related article suggestions

### **Sharing & Export**
- 📧 Email sharing
- 🐦 Social media sharing
- 📤 Export to read-later services (Pocket, Instapaper)
- 🔗 Public article sharing (optional)

### **Advanced Organization**
- 🔍 Smart filters (by keyword, source, date)
- ⭐ Favorite articles
- 📚 Reading lists/collections
- 🎯 Priority feeds

### **User Experience**
- 🔊 Text-to-speech for articles
- 📱 Better mobile experience
- ⚡ Performance optimizations
- 🔔 Basic notifications

## ❌ Intentionally Excluded Features

### **High Maintenance Features**
- ❌ AI/ML features (content analysis, recommendations)
- ❌ Team collaboration and sharing
- ❌ Advanced integrations (Zapier, IFTTT)
- ❌ Native mobile apps
- ❌ Real-time notifications
- ❌ Advanced analytics and reporting

### **Enterprise Features**
- ❌ Threat intelligence
- ❌ Market intelligence
- ❌ Custom AI models
- ❌ SSO integration
- ❌ Multi-tenant architecture

### **Complex Social Features**
- ❌ User comments and discussions
- ❌ Community features
- ❌ Content recommendation algorithms
- ❌ Social media integration beyond basic sharing

## 🔧 Maintenance Strategy

### **Automated Tasks**
- Feed refresh every 15-30 minutes
- Article cleanup (delete articles older than 30 days)
- Database VACUUM operations
- Log rotation
- Security updates via Watchtower
- Health check monitoring

### **Manual Tasks (Monthly)**
- Review error logs (~10 minutes)
- Check disk usage (~5 minutes)
- Verify backup integrity (~5 minutes)
- Update dependencies (~10 minutes)

**Total maintenance time: ~30 minutes/month**

## 📊 Success Metrics

### **Performance Targets**
- Page load time: < 2 seconds
- Feed refresh time: < 10 seconds for 100 feeds
- Memory usage: < 512MB
- Storage growth: < 1GB/month for moderate usage

### **Functionality Targets**
- Support 500+ RSS feeds
- Handle 10,000+ articles in database
- 99.9% uptime
- Zero data loss

## 🚀 Implementation Phases

### **Phase 1: Foundation (Weeks 1-4)**
1. **Week 1**: Project setup, backend API structure
2. **Week 2**: RSS parsing and feed management
3. **Week 3**: Frontend UI and basic reading interface
4. **Week 4**: Organization features and search

### **Phase 2: Polish (Weeks 5-6)**
1. **Week 5**: Mobile responsiveness, PWA features
2. **Week 6**: Performance optimization, deployment setup

### **Phase 3: Enhancement (Weeks 7-8)**
1. **Week 7**: Advanced features from Phase 2 list
2. **Week 8**: Testing, documentation, final deployment

## 📝 Technical Specifications

### **Database Schema**
```sql
-- Core tables
feeds (id, url, title, description, folder_id, created_at, updated_at, last_fetch)
folders (id, name, parent_id, position, created_at)
articles (id, feed_id, title, content, url, published_at, read, saved, created_at)
settings (key, value)
```

### **API Endpoints**
```
GET    /api/feeds              # List all feeds
POST   /api/feeds              # Add new feed
PUT    /api/feeds/:id          # Update feed
DELETE /api/feeds/:id          # Delete feed

GET    /api/articles           # List articles (with filters)
PUT    /api/articles/:id/read  # Mark as read/unread
PUT    /api/articles/:id/save  # Save for later

GET    /api/folders            # List folders
POST   /api/folders            # Create folder
PUT    /api/folders/:id        # Update folder
DELETE /api/folders/:id        # Delete folder

GET    /api/search?q=term      # Search articles
POST   /api/opml/import        # Import OPML
GET    /api/opml/export        # Export OPML
```

### **Configuration**
```yaml
# config.yaml
server:
  port: 8080
  host: "0.0.0.0"

database:
  path: "./data/myfeed.db"

feeds:
  refresh_interval: "15m"
  max_articles_per_feed: 1000
  cleanup_after_days: 30

ui:
  title: "MyFeed"
  theme: "auto"
  articles_per_page: 50
```

This project plan provides a clear roadmap for building a maintainable, feature-rich RSS reader that delivers core Feedly functionality without the complexity overhead.