<div align="center">
  <img src=".github/banner.svg" width="100%" alt="GO-API" style="max-width: 800px; height: auto;" />
  <br />
  <p>
    <img src="https://img.shields.io/badge/made%20by-Thales%20Macena-2D325E?labelColor=F0DB4F&style=for-the-badge&logo=visual-studio-code&logoColor=2D325E" alt="Made by Thales Macena">
    <img alt="Top Language" src="https://img.shields.io/github/languages/top/thalesmacena/go-api?color=2D325E&labelColor=F0DB4F&style=for-the-badge&logo=go&logoColor=2D325E">
    <a href="https://github.com/thalesmacena/go-api/commits/main">
      <img alt="Last Commits" src="https://img.shields.io/github/last-commit/thalesmacena/go-api?color=2D325E&labelColor=F0DB4F&style=for-the-badge&logo=github&logoColor=2D325E">
    </a>
    <a href="https://github.com/thalesmacena/go-api/issues">
      <img alt="Issues" src="https://img.shields.io/github/issues-raw/thalesmacena/go-api?color=2D325E&labelColor=F0DB4F&style=for-the-badge&logo=github&logoColor=2D325E">
    </a>
  </p>
</div>

<div align="center">

# GO-API

**A modern Go REST API built with Echo framework, featuring URL shortening, weather services, and health monitoring capabilities.**

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=for-the-badge&logo=go)](https://golang.org/)
[![Echo Framework](https://img.shields.io/badge/Echo-v4-FF6B6B?style=for-the-badge)](https://echo.labstack.com/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-13+-336791?style=for-the-badge&logo=postgresql)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7+-DC382D?style=for-the-badge&logo=redis)](https://redis.io/)
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?style=for-the-badge&logo=docker)](https://www.docker.com/)
[![AWS](https://img.shields.io/badge/AWS-SQS-FF9900?style=for-the-badge&logo=amazon-aws)](https://aws.amazon.com/)

</div>

---

## 📋 Table of Contents

- [GO-API](#go-api)
  - [📋 Table of Contents](#-table-of-contents)
  - [🚀 Features](#-features)
    - [Redis Package Highlights](#redis-package-highlights)
  - [🛠️ Tech Stack](#️-tech-stack)
  - [📋 Prerequisites](#-prerequisites)
  - [🏃‍♂️ Quick Start](#️-quick-start)
    - [Using Docker Compose (Recommended)](#using-docker-compose-recommended)
    - [Local Development](#local-development)
    - [Redis Examples](#redis-examples)
  - [🔧 Configuration](#-configuration)
    - [Environment Variables](#environment-variables)
  - [📚 API Documentation](#-api-documentation)
    - [Swagger UI](#swagger-ui)
    - [Building Swagger Documentation](#building-swagger-documentation)
  - [🏗️ Project Structure](#️-project-structure)
  - [🔄 Development Workflow](#-development-workflow)
    - [Running Tests](#running-tests)
    - [Building the Application](#building-the-application)
    - [Database Migrations](#database-migrations)
    - [Queue Processing](#queue-processing)
  - [🐳 Docker Usage](#-docker-usage)
    - [Useful Commands](#useful-commands)
    - [Monitoring \& Logging](#monitoring--logging)
  - [🤝 Contributing](#-contributing)
  - [📄 License](#-license)

---

## 🚀 Features

- **Health Check**: System health monitoring with database and queue status
- **URL Shortener**: Create and manage short URLs with automatic cleanup
- **Weather Service**: Asynchronous weather data processing using AWS SQS
- **Redis (Cache, Lock, Pub/Sub)**: High-performance cache with per-cache TTL, distributed locks with auto-refresh, and namespaced Pub/Sub with concurrent workers and auto-reconnect
- **Clean Architecture**: Domain-driven design with clear separation of concerns
- **Database Support**: PostgreSQL with GORM and SQLC
- **AWS Integration**: LocalStack for local development with SQS, S3, DynamoDB
- **Scheduled Tasks**: Automated cleanup and maintenance jobs
- **Request Logging**: Comprehensive request/response logging middleware

### Redis Package Highlights

- Client with fluent configuration: `NewRedisConfig().WithHost(...).WithPoolSize(...)`
- Cache with per-cache TTL: `WithCacheTTL("user_cache", 2*time.Hour)` and default TTL
- Namespaced cache keys: `CacheName::cacheKey`
- Distributed locks with auto-refresh and namespacing: `LockNamespace::lockKey`
- Pub/Sub with namespaced channels, concurrent workers and auto-reconnect
- Health checks for Redis client and Pub/Sub

## 🛠️ Tech Stack

- **Language**: Go 1.24
- **Framework**: Echo v4
- **Database**: PostgreSQL with GORM and SQLC
- **Cache**: Redis 7+ for high-performance caching
- **Queue**: AWS SQS (LocalStack for local development)
- **Architecture**: Clean Architecture with Domain-Driven Design
- **Configuration**: Viper with YAML configuration
- **Logging**: Uber Zap
- **Containerization**: Docker & Docker Compose
- **Documentation**: Swagger/OpenAPI 3.0

## 📋 Prerequisites

- [Go 1.24+](https://golang.org/doc/install)
- [Docker](https://docs.docker.com/get-docker/)
- [Docker Compose](https://docs.docker.com/compose/install/)

## 🏃‍♂️ Quick Start

### Using Docker Compose (Recommended)

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd go-api
   ```

2. **Start all services**
   ```powershell
   docker compose up -d
   ```

3. **Verify the application is running**
   ```powershell
   curl http://localhost:8080/go-api/health
   ```

4. **Stop and cleanup (removing local images)**
   ```powershell
   docker compose down --rmi local
   ```

### Local Development

1. **Install dependencies**
   ```bash
   go mod download
   ```

2. **Start infrastructure services only**
   ```powershell
   docker compose up -d postgres localstack redis
   ```

3. **Run the application**
   ```bash
   go run cmd/go-api/main.go
   ```

### Redis Examples

Run focused Redis examples:

```bash
# Main Redis example (configuration, cache, JSON, batch, scan)
go run example/redis/main.go

# Distributed Lock example
go run example/redis/lock/main.go

# Pub/Sub example
go run example/redis/pubsub/main.go
```

## 🔧 Configuration

The application uses environment variables and YAML configuration. Key settings:

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `SERVER_PORT` | `8080` | Server port |
| `DB_HOST` | `localhost` | Database host |
| `DB_PORT` | `5432` | Database port |
| `DB_USERNAME` | `postgres` | Database username |
| `DB_PASSWORD` | `postgres` | Database password |
| `DB_DATABASE` | `postgres` | Database name |
| `DB_SCHEMA` | `go` | Database schema |
| `REDIS_HOST` | `localhost` | Redis host |
| `REDIS_PORT` | `6379` | Redis port |
| `REDIS_PASSWORD` | `redis_password` | Redis password |
| `REDIS_DB` | `0` | Redis database number |
| `REDIS_POOL_SIZE` | `10` | Redis connection pool size |
| `REDIS_MIN_IDLE_CONNS` | `5` | Redis min idle connections |
| `REDIS_MAX_IDLE_CONNS` | `10` | Redis max idle connections |
| `REDIS_MAX_ACTIVE` | `100` | Redis max active connections |
| `AWS_ENDPOINT` | `http://localhost:4566` | AWS endpoint (LocalStack) |
| `AWS_ACCESS_KEY_ID` | `test` | AWS access key |
| `AWS_SECRET_ACCESS_KEY` | `test` | AWS secret key |

Configuration is managed in `configs/application.yml`. See the file for detailed settings.

## 📚 API Documentation

### Swagger UI

When the application is running, you can access the interactive API documentation at:

- **Swagger UI**: http://localhost:8080/swagger/index.html

The Swagger documentation provides:
- Complete API endpoint documentation
- Request/response schemas
- Interactive testing interface
- Model definitions

### Building Swagger Documentation

To generate or update the Swagger documentation without using Docker Compose:

1. **Install swag CLI**
   ```bash
   go install github.com/swaggo/swag/cmd/swag@latest
   ```

2. **Generate documentation**
   ```bash
   swag init -g cmd/go-api/main.go -o docs
   ```

The generated documentation files will be placed in the `docs/` directory:
- `docs/swagger.json` - JSON format
- `docs/swagger.yaml` - YAML format  
- `docs/docs.go` - Go code for embedding

## 🏗️ Project Structure

```
.
├── cmd/                    # Application entry points
│   ├── go-api/            # Main API application
│   ├── channel/           # Channel processing service
│   └── pagination/        # Pagination example service
├── internal/
│   ├── application/       # Application layer
│   │   ├── controller/    # HTTP controllers
│   │   ├── middleware/    # Custom middleware
│   │   ├── processor/     # Queue processors
│   │   └── schedule/      # Scheduled tasks
│   ├── domain/           # Domain layer
│   │   ├── entity/       # Domain entities
│   │   ├── gateway/      # Interface definitions
│   │   ├── model/        # Domain models & DTOs
│   │   └── usecase/      # Business logic
│   └── infra/            # Infrastructure layer
│       ├── aws/          # AWS implementations
│       └── database/     # Database implementations
├── pkg/                  # Shared packages
│   ├── http/            # HTTP utilities
│   ├── log/             # Logging utilities
│   ├── msg/             # Message handling
│   ├── redis/           # Redis package (client, cache, lock, pubsub, health, config)
│   ├── resource/        # Resource management
│   ├── sqs/             # SQS utilities
│   └── util/            # General utilities
├── configs/              # Configuration files
│   ├── application.yml   # Main application config
│   └── messages.yml      # Message templates
├── docs/                 # Swagger documentation
│   ├── docs.go          # Generated Go documentation
│   ├── swagger.json     # JSON API specification
│   └── swagger.yaml     # YAML API specification
├── opt/                  # Configuration and initialization
│   ├── dump.sql         # Database schema and data
│   ├── redis.conf       # Redis configuration
│   ├── nginx.conf.template # NGINX configuration template
│   ├── init-nginx.sh    # NGINX initialization script
│   └── ready.d/         # LocalStack initialization scripts
│       └── 01-setup-sqs.sh
├── example/              # Usage examples
│   ├── http/            # HTTP client examples
│   ├── log/             # Logging examples
│   ├── msg/             # Message handling examples
│   ├── redis/           # Redis examples (main, pubsub, lock)
│   ├── resource/        # Resource management examples
│   └── sqs/             # SQS examples
├── scripts/              # Build and deployment scripts
├── docker-compose.yml    # Docker services orchestration
├── Dockerfile           # Main application container
├── database.dockerfile  # Database container with custom setup
├── go.mod              # Go module dependencies
└── go.sum              # Go module checksums
```

## 🔄 Development Workflow

### Running Tests
```bash
go test ./...
```

### Building the Application
```bash
go build -o bin/go-api cmd/go-api/main.go
```

### Database Migrations
Database setup is handled automatically through Docker initialization scripts in the `opt/` directory.

### Queue Processing
The application includes SQS workers for asynchronous processing:
- Weather data processing
- Configurable batch sizes and worker pools

## 🐳 Docker Usage

The `docker-compose.yml` includes:

- **postgres**: PostgreSQL database with health checks
- **redis**: Redis cache server with custom configuration (in-memory)
- **localstack**: AWS services emulator (SQS, S3, DynamoDB, Lambda)
- **nginx**: Load balancer for multiple go-api instances
- **go-api-1, go-api-2**: Multiple application instances for load balancing

### Useful Commands

```powershell
# Start all services
docker compose up -d

# View logs
docker compose logs -f go-api

# Stop services and remove local images
docker compose down --rmi local

# Stop services and remove everything (images, volumes, networks)
docker compose down --rmi all --volumes --remove-orphans

# Rebuild and restart
docker compose up -d --build
```

### Monitoring & Logging

- **Health Checks**: Built-in health monitoring for database, Redis (client and Pub/Sub), and queue connections
- **Request Logging**: All HTTP requests are logged with detailed information
- **Structured Logging**: Uses Uber Zap for structured, high-performance logging

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

<div align="center">

**Made with ❤️ by [Thales Macena](https://github.com/thalesmacena)**

<img src="https://img.shields.io/badge/⭐_Star_this_repository-if_it_helped_you!-yellow?style=for-the-badge" alt="Star this repository"/>

</div>