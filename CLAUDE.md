# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

ODI is a privacy-focused document digitization system for scanning, OCR processing, and indexing paper documents. It uses OpenSearch for search, supports local filesystem or encrypted Backblaze B2 storage, and performs OCR via a remote Android-based ML Kit server.

## Build & Development Commands

```bash
# Install dependencies
go mod download

# Build
go build ./...

# Run tests
go test ./...

# Run a single test
go test ./pkg/crypt -run TestEncryptDecrypt

# Run E2E tests (requires E2E_TEST env var)
E2E_TEST=1 go test ./...

# Generate protobuf files
docker run --rm -v $(pwd):/workspace -w /workspace bufbuild/buf:latest generate

# Lint (requires golangci-lint)
golangci-lint run

# Start local OpenSearch
docker-compose up -d
```

**Prerequisites**: OpenCV must be installed for gocv (used for image processing). On Linux: `git clone https://github.com/hybridgroup/gocv && cd gocv && make install`

## CLI Commands

The application uses Cobra CLI with a single entry point. Run with `go run .` or build and execute:

- `serve` - Start REST API server
- `ingest` - Ingest documents from AirScan network scanners
- `index` - Index documents from a local directory
- `reindex` - Reindex documents in OpenSearch
- `ocr` - Process documents through OCR
- `ocrtext` - Extract OCR text from documents
- `decrypt` - Decrypt encrypted documents
- `version` - Show version

## Architecture

### Package Structure

- `internal/cli/` - CLI command handlers using Cobra, configuration binding via Viper
- `internal/config/` - Configuration management wrapper
- `pkg/ingestor/` - Document ingestion pipeline from AirScan scanners
- `pkg/indexer/` - OCR processing, text/metadata extraction, OpenSearch indexing
- `pkg/ocrclient/` - HTTP client for remote OCR service
- `pkg/storage/` - Pluggable storage backends (interface in `model/interface.go`)
  - `b2/` - Backblaze B2 with AES-256 GCM encryption
  - `fs/` - Local filesystem
  - `rclone/` - Rclone integration
- `pkg/crypt/` - AES-256 GCM encryption with PBKDF2 key derivation
- `pkg/zefix/` - Swiss company matching via Zefix database
- `pkg/models/` - Data models (Document, ScannedPage, Barcode)
- `pkg/server/` - REST API using Gin framework

### Data Flow

```
Scanner (AirScan) → Ingestor → Indexer → OpenSearch
                                    ↓
                              OCR Service
                                    ↓
                              Storage (B2/FS)
```

### Configuration

Environment variables (legacy names without prefix also supported):
- `OPENSEARCH_ADDR`, `OPENSEARCH_USERNAME`, `OPENSEARCH_PASSWORD`, `OPENSEARCH_SKIP_TLS`, `OPENSEARCH_INDEX`
- `STORAGE_TYPE` (b2 or filesystem), `B2_ACCOUNT`, `B2_KEY`, `B2_BUCKET_NAME`, `B2_PASSPHRASE`
- `FS_PATH` for filesystem storage
- `OCR_API_ADDR`, `OCR_API_CA_PATH`
- `ZEFIX_DSN` - PostgreSQL connection for Zefix company data
- `SCANNER_NAME` - AirScan scanner hostname

Credentials can use `keychain:` prefix for secure storage (e.g., `B2_KEY=keychain:b2-key`).

## Testing Patterns

- Standard Go testing with `testify` assertions
- HTTP mocking with `gock` for external API tests
- E2E tests gated by `E2E_TEST` environment variable
- Test data in `resources/testdata/`
