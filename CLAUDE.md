# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

A CLI tool that scans documents from SANE network scanners with optional upload to Paperless-ngx. The application:
- Triggers scans via SANE network protocol using `scanimage` or eSCL (AirScan) protocol
- Supports multiple output formats: PDF, PNG, JPG/JPEG (300 DPI, color)
- Optionally uploads documents to Paperless-ngx via REST API (with `--upload-to-paperless` flag)
- Optionally saves scans locally (with `-output` flag)
- Automatically cleans up temporary files when not saving locally

## Development Environment

This project uses **devenv** (https://devenv.sh/) with Nix for managing the development environment. The environment is automatically activated via direnv.

### Environment Setup

- The project uses devenv with Nix flakes
- Go language support is enabled
- Python tooling (uv) is available for scripting/utilities
- Development environment configuration is in `devenv.nix` and `devenv.local.nix`
- Local customizations should go in `devenv.local.nix` (gitignored)

### Configuration

The project supports configuration via both environment variables and command-line flags:

**Environment Variables (`.env` file)**:
- `SCANNER_URL`: URL of the scanner device/service (required)
- `PAPERLESS_URL`: URL of the Paperless instance (required only when using `--upload-to-paperless`)
- `PAPERLESS_TOKEN`: Authentication token for Paperless API access (required only when using `--upload-to-paperless`)

**Command-Line Flags** (override environment variables):
- `--scanner_url`: Override SCANNER_URL
- `--paperless_url`: Override PAPERLESS_URL
- `--paperless_token`: Override PAPERLESS_TOKEN

**Priority**: Command-line flags take precedence over environment variables.

### Common Commands

```bash
# Build the project
cd src && go build -o ../paperless-scanner && cd ..

# Or from root directory
go build -C src -o ../paperless-scanner

# Scan and upload to Paperless (using .env config)
./paperless-scanner --upload-to-paperless

# Save scan locally (PDF)
./paperless-scanner -output /path/to/document.pdf

# Save scan locally (PNG)
./paperless-scanner -output /path/to/document.png

# Save scan locally AND upload to Paperless
./paperless-scanner -output /path/to/document.pdf --upload-to-paperless

# Use command-line flags instead of .env
./paperless-scanner --scanner_url http://192.168.1.100 -output scan.pdf

# Upload with all config via flags
./paperless-scanner \
  --scanner_url http://192.168.1.100 \
  --paperless_url https://paperless.example.com \
  --paperless_token mytoken123 \
  --upload-to-paperless

# Override only scanner URL
./paperless-scanner --scanner_url http://192.168.1.200 --upload-to-paperless

# Run with verbose logging
./paperless-scanner --upload-to-paperless -verbose

# Run tests
go test ./...

# Run a single test
go test -run TestName ./path/to/package

# Format code
cd src && go fmt ./... && cd ..

# Docker commands
docker build -t paperless-scanner .
docker run --rm --network host -e SCANNER_URL=http://192.168.1.100 paperless-scanner -output /scans/scan.pdf
docker-compose build
docker-compose up
```

### devenv Integration

```bash
# Enter the development shell (if not using direnv)
devenv shell

# Run devenv tests
devenv test

# Update dependencies
devenv update
```

## Architecture Notes

### Custom Claude Code Agents

This project has custom Claude Code agents configured in `devenv.local.nix`:

- **code-reviewer**: Proactive code review agent that runs after code changes
  - Reviews for quality, security, and maintainability
  - Has access to: Read, Grep, TodoWrite tools

- **debugger**: Debugging specialist for errors and test failures
  - Root cause analysis and minimal fixes
  - Has access to: Read, Edit, Bash, Grep, Glob tools

- **test-writer**: Test suite specialist
  - Writes comprehensive tests following project conventions
  - Never modifies production code when fixing tests
  - Has access to: Read, Write, Edit, Bash tools

### MCP Server

The project includes a devenv MCP (Model Context Protocol) server configured for enhanced Claude Code integration.

## Project Structure

```
.
├── src/                 # Source code directory
│   ├── main.go          # CLI entry point, config loading, orchestration
│   ├── scanner.go       # SANE scanner client (uses scanimage command)
│   ├── escl_scanner.go  # eSCL (AirScan) scanner client (HTTP-based)
│   ├── paperless.go     # Paperless-ngx API client
│   ├── go.mod           # Go module definition
│   └── go.sum           # Go module checksums
├── Dockerfile           # Multi-stage Docker build
├── docker-compose.yml   # Docker Compose configuration
├── .dockerignore        # Docker build exclusions
├── .env                 # Environment configuration (gitignored)
├── .env.example         # Environment configuration template
├── README.md            # User documentation
└── CLAUDE.md            # Development documentation
```

### Key Components

**main.go**:
- Loads configuration from `.env` file with command-line flag overrides
- Provides CLI flags:
  - `-output`: Save scan to specified path
  - `--upload-to-paperless`: Upload to Paperless-ngx
  - `-verbose`: Enable verbose logging
  - `--scanner_url`: Override SCANNER_URL env var
  - `--paperless_url`: Override PAPERLESS_URL env var
  - `--paperless_token`: Override PAPERLESS_TOKEN env var
- Orchestrates the scan workflow with optional upload
- Handles cleanup of temporary files (when not saving locally)
- Determines output format from file extension

**scanner.go**:
- `Scanner` type wraps SANE scanner interactions
- Uses `scanimage` command-line tool via `exec.Command`
- Supports multiple formats: PDF, PNG, JPEG (300 DPI, color)
- Saves scans to `/tmp/paperless-scanner/scan-TIMESTAMP.<ext>`

**escl_scanner.go**:
- `ESCLScanner` type wraps eSCL (AirScan) protocol interactions
- Uses HTTP REST API for scanner communication
- Supports multiple formats: PDF, PNG, JPEG (300 DPI, color)
- Used when SCANNER_URL is an HTTP/HTTPS URL

**paperless.go**:
- `PaperlessClient` handles Paperless-ngx API communication
- Uploads documents via POST to `/api/documents/post_document/`
- Uses token-based authentication
- Returns document ID on successful upload

## Implementation Details

### SANE Scanner Integration

The scanner client uses the `scanimage` command (SANE command-line tool) to interact with network scanners:
- Device format: `net:<IP_ADDRESS>` (e.g., `net:192.168.3.11`)
- Requires SANE backends to be installed on the system
- macOS: `brew install sane-backends`
- Linux: `sudo apt install sane-utils`

### Paperless-ngx API

Documents are uploaded using the REST API:
- Endpoint: `POST /api/documents/post_document/`
- Authentication: `Authorization: Token <PAPERLESS_TOKEN>`
- Content-Type: `multipart/form-data`
- Field name: `document`

### Error Handling

The application handles common failure scenarios:
- Missing or invalid configuration
- Scanner unavailable or scan failure
- Paperless server unreachable or authentication failure
- File I/O errors

### Dependencies

- `github.com/joho/godotenv` - Environment variable loading from `.env`
