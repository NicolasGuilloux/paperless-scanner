# Paperless Scanner

A CLI tool to scan documents from SANE network scanners and automatically upload them to Paperless-ngx.

## Features

- Scan documents from SANE network scanners
- Automatic upload to Paperless-ngx
- Optional local file saving
- Multiple output formats: PDF, PNG, JPG/JPEG (300 DPI, color)
- Automatic cleanup of temporary files

## Prerequisites

### For Native Installation

- Go 1.22 or later
- `scanimage` command (part of SANE package)
  - macOS: `brew install sane-backends`
  - Linux: `sudo apt install sane-utils` (Debian/Ubuntu) or `sudo yum install sane-backends` (RHEL/Fedora)
- SANE network scanner accessible on your network
- Paperless-ngx instance with API access (optional, only if uploading)

### For Docker Installation

- Docker or Docker Compose
- SANE network scanner accessible on your network
- Paperless-ngx instance with API access (optional, only if uploading)

## Installation

### Native Installation

```bash
cd src
go build -o ../paperless-scanner
cd ..
```

Or from the root directory:

```bash
go build -C src -o ../paperless-scanner
```

### Docker Installation

The Docker image is publicly available on Docker Hub at `nicolasguilloux/paperless-scanner:latest`.

Pull the pre-built image:

```bash
docker pull nicolasguilloux/paperless-scanner:latest
```

Or build the Docker image locally:

```bash
docker build -t paperless-scanner .
```

Or use Docker Compose:

```bash
docker-compose build
```

## Configuration

You can configure the scanner using either environment variables (via `.env` file) or command-line flags. Command-line flags take precedence over environment variables.

### Using Environment Variables (.env file)

Create a `.env` file in the project root with the following variables:

```env
SCANNER_URL=192.168.1.100
PAPERLESS_URL=https://paperless.example.com  # Only required if using --upload-to-paperless
PAPERLESS_TOKEN=your_api_token_here          # Only required if using --upload-to-paperless
```

### Using Command-Line Flags

You can override the `.env` file using command-line flags:

- `--scanner_url`: IP address or hostname of your SANE network scanner (required)
- `--paperless_url`: Base URL of your Paperless-ngx instance (required only when using `--upload-to-paperless`)
- `--paperless_token`: API token for authentication (required only when using `--upload-to-paperless`)

### Configuration Priority

1. Command-line flags (highest priority)
2. Environment variables from `.env` file (fallback)

### Configuration Variables

- `SCANNER_URL` / `--scanner_url`: IP address or hostname of your SANE network scanner (required)
- `PAPERLESS_URL` / `--paperless_url`: Base URL of your Paperless-ngx instance (required only when using `--upload-to-paperless`)
- `PAPERLESS_TOKEN` / `--paperless_token`: API token for authentication (required only when using `--upload-to-paperless`, found in Paperless settings)

## Usage

### Scan and upload to Paperless

```bash
./paperless-scanner --upload-to-paperless
```

This will:
1. Scan a document from the configured scanner
2. Save it temporarily as a PDF
3. Upload it to Paperless-ngx
4. Delete the temporary file

### Save scan locally

The output format is automatically determined by the file extension:

```bash
# Save as PDF
./paperless-scanner -output /path/to/save/document.pdf

# Save as PNG
./paperless-scanner -output /path/to/save/document.png

# Save as JPEG
./paperless-scanner -output /path/to/save/document.jpg
```

Supported formats: `.pdf`, `.png`, `.jpg`, `.jpeg`

### Save locally AND upload to Paperless

You can combine both flags:

```bash
./paperless-scanner -output /path/to/save/document.pdf --upload-to-paperless
```

This will save the scan locally and also upload it to Paperless.

### Enable verbose logging

```bash
./paperless-scanner --upload-to-paperless -verbose
```

### Using command-line flags instead of .env

You can use command-line flags to override or replace the `.env` file:

```bash
# Use flags for all configuration
./paperless-scanner --scanner_url http://192.168.1.100 -output scan.pdf

# Upload to Paperless with flags
./paperless-scanner \
  --scanner_url http://192.168.1.100 \
  --paperless_url https://paperless.example.com \
  --paperless_token mytoken123 \
  --upload-to-paperless

# Override only scanner URL, use .env for Paperless config
./paperless-scanner --scanner_url http://192.168.1.200 --upload-to-paperless
```

### Running with Docker

#### Using Docker Run

```bash
# Scan and save to local directory
docker run --rm \
  --network host \
  -v $(pwd)/scans:/scans \
  -e SCANNER_URL=http://192.168.1.100 \
  nicolasguilloux/paperless-scanner:latest \
  -output /scans/scan.pdf

# Scan and upload to Paperless
docker run --rm \
  --network host \
  -e SCANNER_URL=http://192.168.1.100 \
  -e PAPERLESS_URL=http://paperless:8000 \
  -e PAPERLESS_TOKEN=your_token_here \
  nicolasguilloux/paperless-scanner:latest \
  --upload-to-paperless

# Scan, save locally, and upload to Paperless
docker run --rm \
  --network host \
  -v $(pwd)/scans:/scans \
  -e SCANNER_URL=http://192.168.1.100 \
  -e PAPERLESS_URL=http://paperless:8000 \
  -e PAPERLESS_TOKEN=your_token_here \
  nicolasguilloux/paperless-scanner:latest \
  -output /scans/scan.pdf --upload-to-paperless

# Use command-line flags instead of environment variables
docker run --rm \
  --network host \
  -v $(pwd)/scans:/scans \
  nicolasguilloux/paperless-scanner:latest \
  --scanner_url http://192.168.1.100 \
  --paperless_url http://paperless:8000 \
  --paperless_token your_token_here \
  -output /scans/scan.png \
  --upload-to-paperless
```

#### Using Docker Compose

1. Edit `docker-compose.yml` to configure your scanner and Paperless settings
2. Run the scanner:

```bash
# Scan and upload to Paperless
docker-compose up

# Scan and save locally
docker-compose run --rm paperless-scanner -output /scans/scan.pdf

# One-time scan with custom settings
docker-compose run --rm paperless-scanner \
  --scanner_url http://192.168.1.200 \
  -output /scans/scan.jpg
```

**Note**: Docker uses `--network host` to allow direct access to network scanners. Sometimes it is not required, but to support the most use cases, you should keep it.

## How It Works

1. **Scan**: Uses `scanimage` to trigger a scan on the SANE network scanner
   - Format: PDF, PNG, or JPEG (determined by output file extension, defaults to PDF)
   - Resolution: 300 DPI
   - Mode: Color

2. **Save**: Saves the scan to `/tmp/paperless-scanner/scan-TIMESTAMP.<ext>` (or to specified path with `-output`)

3. **Upload**: Sends the document to Paperless-ngx via the `/api/documents/post_document/` endpoint (when `--upload-to-paperless` flag is specified)

4. **Cleanup**: Removes the temporary file after completion (unless saved locally with `-output`)

## Troubleshooting

### Scanner not found

```bash
# List available SANE devices
scanimage -L

# Test direct scan
scanimage --device-name net:192.168.1.100 --format=pdf > test.pdf
```

### Paperless upload fails

- Verify your `PAPERLESS_TOKEN` is correct
- Check that the Paperless URL is accessible
- Ensure your Paperless instance accepts API uploads

### Permission errors

The scanner may require specific permissions. Make sure the SANE daemon is running and accessible.

## Development

This project uses devenv for the development environment. See `CLAUDE.md` for details.

```bash
# Run with go run
go run . -verbose

# Build
go build -o paperless-scanner

# Run tests
go test ./...
```
