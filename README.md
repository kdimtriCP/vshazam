# VShazam

A video-based film identification service built with Go and HTMX.

## Overview

VShazam is a proof-of-concept web service that recognizes films from short user-supplied video clips. This project implements a "Shazam for videos" using AI services and web search APIs.

## Technology Stack

- **Backend**: Go 1.24.5
- **Web Framework**: Chi router v5
- **Frontend**: Server-rendered HTML + HTMX
- **Database**: SQLite (Stage 2), PostgreSQL (Stage 4+)
- **AI Services**: GPT-4o Vision, Google Vision API (Stage 5+)

## Getting Started

### Prerequisites

- Go 1.24.5 or higher
- Git

### Installation

1. Clone the repository:
```bash
git clone https://github.com/kdimtricp/vshazam.git
cd vshazam
```

2. Install dependencies:
```bash
go mod download
```

3. Copy environment variables (optional):
```bash
cp .env.example .env
# Edit .env as needed
```

4. Set environment variables (optional):
```bash
export PORT=8080                      # Server port (default: 8080)
export MAX_UPLOAD_SIZE=104857600      # Max upload size in bytes (default: 100MB)
export UPLOAD_DIR=./uploads           # Upload directory (default: ./uploads)
export DB_PATH=./vshazam.db          # SQLite database path (default: ./vshazam.db)
```

### Running the Application

**With Docker (Recommended):**
```bash
# Build and start all services
make docker-up

# View logs
make docker-logs

# Stop services
make docker-down
```

**Without Docker:**
```bash
# Using SQLite (default)
go run cmd/server/main.go

# Using PostgreSQL
export DB_TYPE=postgres
export DB_HOST=localhost
export DB_PORT=5432
export DB_USER=vshazam
export DB_PASSWORD=vshazam_dev
export DB_NAME=vshazam
go run cmd/server/main.go
```

**Using Make:**
```bash
make run
```

### Testing the Application

Test the health check endpoint:
```bash
curl http://localhost:8080/ping
# Expected response: pong
```

Open the web interface:
```bash
open http://localhost:8080
```

Upload a video:
```bash
open http://localhost:8080/upload
```

## Project Structure

```
vshazam/
├── cmd/
│   └── server/
│       └── main.go            # Application entry point
├── internal/
│   ├── api/                   # HTTP API
│   │   ├── router.go          # Route definitions
│   │   └── handlers.go        # Request handlers
│   ├── database/              # Database layer
│   │   ├── db.go             # Database connection
│   │   └── video_repo.go     # Video repository
│   ├── models/                # Data models
│   │   └── video.go          # Video model
│   └── storage/               # File storage
│       ├── storage.go        # Storage interface
│       └── local_storage.go  # Local filesystem implementation
├── web/
│   ├── templates/             # HTML templates
│   │   ├── base.html          # Base layout
│   │   ├── upload.html        # Upload page
│   │   └── _video_item.html   # Video list item partial
│   └── static/                # Static assets
│       └── styles.css         # CSS styles
├── uploads/                   # Uploaded video files
├── go.mod                     # Go module file
├── go.sum                     # Go dependencies
├── README.md                  # This file
├── .env.example               # Environment variables example
├── Makefile                   # Build commands
└── vshazam.db                # SQLite database
```

## Development

### Building the Application

```bash
make build
```

### Running Tests

```bash
# Run all tests
make test

# Run unit tests only
make test-unit

# Run integration tests
make test-integration

# Run tests with coverage
make test-coverage
```

## Implementation Stages

1. **Stage 1**: Basic HTTP server with routing ✓
2. **Stage 2**: File upload with HTMX and local storage ✓
3. **Stage 3**: Video playback with Range request support ✓
4. **Stage 4**: Docker containerization, PostgreSQL, and full-text search ✓
5. **Stage 5**: AI service integration
6. **Stage 6**: Film identification algorithm
7. **Stage 7**: Cloud deployment

## Contributing

This is a proof-of-concept project for learning purposes.

## License

This project is for educational purposes only.