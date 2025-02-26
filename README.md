# Go Redis API Caching Service

This project demonstrates a simple API built with Golang that uses Redis to cache request/response pairs. Identical requests will be served from the cache instead of being reprocessed.

## Features

- Go API with JSON request/response
- Redis caching of requests
- Docker and Docker Compose setup
- Cache expiration (1 hour by default)
- Health check endpoint

## Project Structure

```
go-redis-api/
├── main.go           # Main application code
├── Dockerfile        # Docker image definition
├── docker-compose.yml # Docker Compose configuration
├── go.mod            # Go module file
└── README.md         # This file
```

## How It Works

1. The API receives a JSON request (e.g., `{"req":"Hello World!"}`)
2. It checks if this exact request is already cached in Redis
3. If found in cache, it returns the cached response
4. If not found, it processes the request, caches the result, and returns the response

## API Endpoints

- `POST /api/process` - Main endpoint that processes requests and handles caching
- `GET /health` - Health check endpoint

## Request/Response Format

**Request:**
```json
{
  "req": "Hello World!"
}
```

**Response:**
```json
{
  "message": "Processed: Hello World!",
  "timestamp": "2025-02-27T15:04:05Z",
  "cached": false
}
```

The `cached` field will be `false` for newly processed requests and `true` for responses served from cache.

## Running the Application

### Prerequisites

- Docker and Docker Compose installed

### Starting the Service

```bash
docker-compose up -d
```

### Testing the API

```bash
# First request (will be processed)
curl -X POST -H "Content-Type: application/json" -d '{"req":"Hello World!"}' http://localhost:8080/api/process

# Second identical request (will be served from cache)
curl -X POST -H "Content-Type: application/json" -d '{"req":"Hello World!"}' http://localhost:8080/api/process
```

### Stopping the Service

```bash
docker-compose down
```

## Configuration

You can modify these settings in the code:

- Cache expiration time (currently set to 1 hour)
- Redis connection parameters
- Port number (currently 8080)