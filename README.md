<div align="center">
  <img src=".github/banner.svg" width="546" alt="GO-API" />
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
[![Docker](https://img.shields.io/badge/Docker-Compose-2496ED?style=for-the-badge&logo=docker)](https://www.docker.com/)
[![AWS](https://img.shields.io/badge/AWS-SQS-FF9900?style=for-the-badge&logo=amazon-aws)](https://aws.amazon.com/)

</div>

---

## 📋 Table of Contents

- [GO-API](#go-api)
  - [📋 Table of Contents](#-table-of-contents)
  - [🚀 Features](#-features)
  - [🛠️ Tech Stack](#️-tech-stack)
  - [📋 Prerequisites](#-prerequisites)
  - [🏃‍♂️ Quick Start](#️-quick-start)
    - [Using Docker Compose (Recommended)](#using-docker-compose-recommended)
    - [Local Development](#local-development)
  - [🔧 Configuration](#-configuration)
    - [Environment Variables](#environment-variables)
    - [Configuration File](#configuration-file)
  - [📚 API Endpoints](#-api-endpoints)
    - [Health Check](#health-check)
    - [URL Shortener](#url-shortener)
    - [Weather Service](#weather-service)
  - [🏗️ Project Structure](#️-project-structure)
  - [🔄 Development Workflow](#-development-workflow)
    - [Running Tests](#running-tests)
    - [Building the Application](#building-the-application)
    - [Database Migrations](#database-migrations)
    - [Queue Processing](#queue-processing)
  - [🐳 Docker Services](#-docker-services)
    - [Useful Docker Commands](#useful-docker-commands)
  - [🔍 Monitoring \& Logging](#-monitoring--logging)
  - [🚀 Production Deployment](#-production-deployment)
  - [🤝 Contributing](#-contributing)
  - [📄 License](#-license)
  - [📞 Support](#-support)
  - [🌐 API Examples](#-api-examples)
    - [Health Check Example](#health-check-example)
    - [URL Shortener Examples](#url-shortener-examples)
    - [Weather Service Example](#weather-service-example)
  - [🎯 Project Statistics](#-project-statistics)
    - [📊 Features Overview](#-features-overview)

---

## 🚀 Features

- **Health Check**: System health monitoring with database and queue status
- **URL Shortener**: Create and manage short URLs with automatic cleanup
- **Weather Service**: Asynchronous weather data processing using AWS SQS
- **Clean Architecture**: Domain-driven design with clear separation of concerns
- **Database Support**: PostgreSQL with GORM and SQLC
- **AWS Integration**: LocalStack for local development with SQS, S3, DynamoDB
- **Scheduled Tasks**: Automated cleanup and maintenance jobs
- **Request Logging**: Comprehensive request/response logging middleware

## 🛠️ Tech Stack

- **Language**: Go 1.24
- **Framework**: Echo v4
- **Database**: PostgreSQL with GORM and SQLC
- **Queue**: AWS SQS (LocalStack for local development)
- **Architecture**: Clean Architecture with Domain-Driven Design
- **Configuration**: Viper with YAML configuration
- **Logging**: Uber Zap
- **Containerization**: Docker & Docker Compose

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
   docker compose up -d postgres localstack
   ```

3. **Run the application**
   ```bash
   go run cmd/go-api/main.go
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
| `AWS_ENDPOINT` | `http://localhost:4566` | AWS endpoint (LocalStack) |
| `AWS_ACCESS_KEY_ID` | `test` | AWS access key |
| `AWS_SECRET_ACCESS_KEY` | `test` | AWS secret key |

### Configuration File

Configuration is managed in `configs/application.yml`. See the file for detailed settings.

## 📚 API Endpoints

### Health Check
- `GET /go-api/health` - System health status

### URL Shortener
- `GET /go-api/short-url` - List all short URLs (with pagination)
- `GET /go-api/short-url/{hash}` - Redirect to original URL
- `POST /go-api/short-url` - Create new short URL
- `PUT /go-api/short-url/{hash}` - Update short URL
- `DELETE /go-api/short-url/{hash}` - Delete short URL

### Weather Service
- `POST /go-api/weather/batch` - Process weather data in batch
- Weather processing is handled asynchronously via SQS queues

## 🏗️ Project Structure

```
.
├── cmd/go-api/              # Application entry point
├── internal/
│   ├── application/         # Application layer
│   │   ├── controller/      # HTTP controllers
│   │   ├── middleware/      # Custom middleware
│   │   ├── processor/       # Queue processors
│   │   └── schedule/        # Scheduled tasks
│   ├── domain/             # Domain layer
│   │   ├── gateway/        # Interface definitions
│   │   ├── model/          # Domain models
│   │   └── usecase/        # Business logic
│   └── infra/              # Infrastructure layer
│       ├── aws/            # AWS implementations
│       └── database/       # Database implementations
├── pkg/                    # Shared packages
├── configs/                # Configuration files
├── database/               # Database migrations/scripts
├── docker-compose.yml      # Docker services
└── Dockerfile             # Application container
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
Database setup is handled automatically through Docker initialization scripts in the `database/` directory.

### Queue Processing
The application includes SQS workers for asynchronous processing:
- Weather data processing
- Configurable batch sizes and worker pools

## 🐳 Docker Services

The `docker-compose.yml` includes:

- **postgres**: PostgreSQL database with health checks
- **localstack**: AWS services emulator (SQS, S3, DynamoDB, Lambda)
- **go-api**: Main application container

### Useful Docker Commands

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

## 🔍 Monitoring & Logging

- **Health Checks**: Built-in health monitoring for database and queue connections
- **Request Logging**: All HTTP requests are logged with detailed information
- **Structured Logging**: Uses Uber Zap for structured, high-performance logging

## 🚀 Production Deployment

For production deployment:

1. Update environment variables in your deployment environment
2. Configure proper AWS credentials and endpoints
3. Set up proper database connection strings
4. Configure appropriate resource limits in Docker containers

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 📞 Support

For support and questions, please open an issue in the repository.

---

## 🌐 API Examples

<details>
<summary><b>Click to expand API usage examples</b></summary>

### Health Check Example

```bash
curl -X GET http://localhost:8080/go-api/health
```

**Response:**
```json
{
  "status": "UP",
  "timestamp": "2024-01-15T10:30:00Z",
  "services": {
    "database": "UP",
    "queue": "UP"
  }
}
```

### URL Shortener Examples

**Create Short URL:**
```bash
curl -X POST http://localhost:8080/go-api/short-url \
  -H "Content-Type: application/json" \
  -d '{
    "url": "https://www.example.com/very/long/url/path",
    "custom_hash": "my-link"
  }'
```

**Get All Short URLs:**
```bash
curl -X GET "http://localhost:8080/go-api/short-url?page=0&size=10"
```

**Redirect via Short URL:**
```bash
curl -X GET http://localhost:8080/go-api/short-url/my-link
```

### Weather Service Example

**Process Weather Batch:**
```bash
curl -X POST http://localhost:8080/go-api/weather/batch \
  -H "Content-Type: application/json" \
  -d '{
    "cities": ["São Paulo", "Rio de Janeiro", "Brasília"],
    "date": "2024-01-15"
  }'
```

</details>

---

<div align="center">

## 🎯 Project Statistics

<table>
<tr>
<td align="center">
<img src="https://img.shields.io/badge/Language-Go-00ADD8?style=flat-square&logo=go" alt="Language"/>
<br/>
<b>Modern Go</b>
</td>
<td align="center">
<img src="https://img.shields.io/badge/Architecture-Clean-brightgreen?style=flat-square" alt="Architecture"/>
<br/>
<b>Clean Architecture</b>
</td>
<td align="center">
<img src="https://img.shields.io/badge/Database-PostgreSQL-336791?style=flat-square&logo=postgresql" alt="Database"/>
<br/>
<b>PostgreSQL</b>
</td>
<td align="center">
<img src="https://img.shields.io/badge/Queue-AWS_SQS-FF9900?style=flat-square&logo=amazon-aws" alt="Queue"/>
<br/>
<b>AWS SQS</b>
</td>
</tr>
</table>

### 📊 Features Overview

<div style="display: flex; justify-content: space-around; margin: 20px 0;">
  <div style="text-align: center; padding: 15px; border: 2px solid #667eea; border-radius: 10px; margin: 5px;">
    <h4 style="color: #667eea; margin: 0;">🔗 URL Shortener</h4>
    <p style="margin: 5px 0; font-size: 14px;">Create, manage, and redirect short URLs with automatic cleanup</p>
  </div>
  <div style="text-align: center; padding: 15px; border: 2px solid #764ba2; border-radius: 10px; margin: 5px;">
    <h4 style="color: #764ba2; margin: 0;">🌤️ Weather Service</h4>
    <p style="margin: 5px 0; font-size: 14px;">Asynchronous weather data processing via SQS queues</p>
  </div>
  <div style="text-align: center; padding: 15px; border: 2px solid #00ADD8; border-radius: 10px; margin: 5px;">
    <h4 style="color: #00ADD8; margin: 0;">💚 Health Monitoring</h4>
    <p style="margin: 5px 0; font-size: 14px;">Real-time system health checks for all services</p>
  </div>
</div>

---

**Made with ❤️ by [Thales Macena](https://github.com/thalesmacena)**

<img src="https://img.shields.io/badge/⭐_Star_this_repository-if_it_helped_you!-yellow?style=for-the-badge" alt="Star this repository"/>

</div>
