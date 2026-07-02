# ShortURL

A high-performance URL shortening service with analytics, role-based access control, and real-time telemetry.

## Features

- **URL Shortening** — Create short URLs with custom or auto-generated slugs
- **Click Analytics** — Track device type, browser, and geographic origin
- **Role-Based Access** — Admin and user roles with Casbin RBAC
- **Redis Caching** — Cache-aside pattern for fast URL lookups
- **Rate Limiting** — Configurable request throttling
- **Admin Panel** — Manage users, URLs, and view system stats
- **Landing Page** — Marketing page with interactive terminal demo
- **Health Check** — `/health` endpoint for monitoring

## Tech Stack

| Component | Technology |
|-----------|------------|
| Backend | Go 1.26, Gin |
| Database | PostgreSQL 15 (PostGIS) |
| Cache | Redis 7 |
| Auth | JWT + Casbin RBAC |
| Frontend | Vanilla JS, Chart.js, Lucide Icons |
| CI/CD | GitHub Actions |
| Registry | GitHub Container Registry (GHCR) |
| DNS/SSL | Cloudflare |
| Secrets | Infisical Cloud |
| Deployment | Docker Compose on VPS |

## Prerequisites

- Go 1.26+
- Docker and Docker Compose
- PostgreSQL 15+ (or use Docker)
- Redis 7+ (or use Docker)

## Local Development

### Quick Start

```bash
# Clone the repository
git clone https://github.com/gopal-chhetri/url-shortener.git
cd url-shortener

# Start all services (app, database, redis)
make build

# Or with Docker Compose directly
cd deployments/local-dev
docker compose up --build
```

The app will be available at `http://localhost:8080`.

### Manual Setup

```bash
# Start PostgreSQL and Redis (or use Docker)
docker compose -f deployments/local-dev/compose.yaml up db redis -d

# Install dependencies
go mod download

# Run database migrations
migrate -database "postgres://postgres:password@localhost:5432/url-shortener?sslmode=disable" -path migrations up

# Generate swagger docs
make swagger

# Start the server
make run
```

### Available Commands

| Command | Description |
|---------|-------------|
| `make run` | Start with hot reload (Air) |
| `make build` | Build and run with Docker Compose |
| `make up` | Start Docker containers |
| `make down` | Stop Docker containers |
| `make migrate` | Run database migrations |
| `make migrate-down` | Rollback migrations |
| `make swagger` | Generate swagger docs |
| `make gen` | Generate SQLC code |
| `make format` | Format Go code |

## Environment Variables

Copy `deployments/local-dev/.env.sample` to `deployments/local-dev/.env` and configure:

```env
# Database
DB_HOST=db
DB_PORT=5432
DB_NAME=url-shortener
DB_USER=postgres
DB_PASS=password

# Application
BASE_URL=http://localhost:8080
PORT=8080
APP_ENV=LOCAL

# Redis
REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# Auth
ACCESS_TOKEN_SECRET=your-secret-key
REFRESH_TOKEN_SECRET=your-refresh-secret
ACCESS_TOKEN_EXPIRY_MINUTE=3600
REFRESH_TOKEN_EXPIRY_MINUTE=10080
```

## API Documentation

Once running, visit:
- **Swagger UI**: `http://localhost:8080/swagger/index.html`
- **Health Check**: `http://localhost:8080/health`

### Key Endpoints

| Method | Path | Description | Auth |
|--------|------|-------------|------|
| POST | `/api/v1/auth/register` | Register new user | No |
| POST | `/api/v1/auth/login` | Login | No |
| POST | `/api/v1/urls` | Create short URL | Yes |
| GET | `/:code` | Redirect to original URL | No |
| GET | `/api/v1/urls` | List user's URLs | Yes |
| DELETE | `/api/v1/urls/:id` | Deactivate URL | Yes |
| GET | `/api/v1/urls/:id/analytics` | Get URL analytics | Yes |

## Deployment

### Production Setup

1. **Set up Infisical** — Create account at [app.infisical.com](https://app.infisical.com), create project and secrets

2. **Configure GitHub Secrets** — Add `VPS_HOST`, `VPS_SSH_USER`, `VPS_SSH_KEY` (or use Infisical OIDC)

3. **Set up Cloudflare** — Add DNS A record pointing to your VPS IP, enable proxy

4. **Deploy to VPS**:

```bash
# On the VPS, install Docker and Infisical CLI
curl -1sLf 'https://artifacts-cli.infisical.com/setup.deb.sh' | sudo -E bash
sudo apt-get install infisical

# Clone and configure
git clone https://github.com/gopal-chhetri/url-shortener.git /opt/url-shortener
cd /opt/url-shortener/deployments/production
cp .env.example .env
# Edit .env with your Infisical credentials

# Deploy
./deploy.sh latest
```

### CI/CD Pipeline

Push to `main` triggers:
1. **CI**: Lint → Test → Build
2. **CD**: Build Docker image → Push to GHCR → Deploy to VPS

### Image Versioning

| Trigger | Tag | Example |
|---------|-----|---------|
| Push to main | `main-<sha>` | `main-abc1234` |
| Git tag | `v1.2.3` | `v1.2.3` |

### Rollback

Automatic rollback on health check failure:
```bash
./deploy.sh main-abc1234  # Deploy specific version
# If health check fails, automatically rolls back
```

## Project Structure

```
url-shortener/
├── cmd/
│   └── url-shortener/
│       └── main.go                 # Entry point
├── internal/
│   ├── admin/                      # Admin module
│   ├── auth/                       # Authentication
│   ├── bootstrap/                  # App initialization
│   ├── db/sqlc/                    # Generated database code
│   ├── health/                     # Health check
│   ├── infra/                      # Infrastructure (config, DB, Redis)
│   ├── middleware/                  # Auth, CORS, rate limiting
│   ├── response/                   # API response helpers
│   ├── routes/                     # Route registration
│   ├── url/                        # URL shortener core
│   └── utils/                      # Utilities
├── migrations/                     # Database migrations
├── web/                            # Frontend (HTML/CSS/JS)
│   ├── index.html                  # Dashboard SPA
│   ├── landing.html                # Landing page
│   ├── app.js                      # Dashboard logic
│   ├── navbar.css                  # Shared navbar styles
│   └── style.css                   # Dashboard styles
├── deployments/
│   ├── local-dev/                  # Docker Compose for development
│   └── production/                 # Production deployment
│       ├── Dockerfile              # Multi-stage build
│       ├── compose.yml             # Production Docker Compose
│       └── deploy.sh               # Deploy + rollback script
├── docs/                           # Swagger documentation
└── .github/workflows/              # GitHub Actions
    ├── ci.yml                      # CI pipeline
    └── deploy.yml                  # CD pipeline
```

## License

Apache License 2.0
