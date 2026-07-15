# Inventory API

A Go inventory management API with product, inventory, and stock tracking.

## Prerequisites

- Go 1.22+
- Docker & Docker Compose (for the database)

## Getting Started

### 1. Start the database

```bash
docker compose up -d
```

This launches a PostgreSQL 16 container with the following defaults:
| Variable     | Default    |
|-------------|------------|
| Host        | localhost  |
| Port        | 5432       |
| User        | postgres   |
| Password    | postgres   |
| Database    | inventory  |

The database and tables are created automatically on first startup.

### 2. Run the application

```bash
go run ./cmd/
```

The server starts on `http://localhost:8080`.

## Configuration

All configuration is via environment variables:

| Variable       | Default     | Description          |
|---------------|-------------|----------------------|
| `DB_HOST`     | `localhost` | Database host        |
| `DB_PORT`     | `5432`      | Database port        |
| `DB_USER`     | `postgres`  | Database user        |
| `DB_PASSWORD` | `postgres`  | Database password    |
| `DB_NAME`     | `inventory` | Database name        |
| `DB_SSLMODE`  | `disable`   | PostgreSQL SSL mode  |
| `PORT`        | `8080`      | HTTP server port     |
