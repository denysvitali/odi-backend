# odi-backend

## Overview

ODI is a software for scanning, OCR processing, and indexing paper documents. This system allows users to digitize their physical documents and make them searchable via OpenSearch.

This repository contains the backend of the system. The frontend can be found [here](https://github.com/denysvitali/odi-frontend).

![ODI Frontend](./docs/odi-frontend.jpg)

## Features

- Document scanning support (via [airscan](https://github.com/stapelberg/airscan/))
- Optical Character Recognition (OCR) (via [ocr-server](https://github.com/denysvitali/ocr-server))
- Document indexing and search (via [OpenSearch](https://opensearch.org/))
- Storage management for digitized documents (local filesystem or Backblaze B2)


## Rationale

Private data should stay private. All the choices in this project are made with privacy in mind. The system is designed to be self-hosted and to keep your data secure.  
The way we perform OCR is by sending the document to be processed on an OCR server which is running locally on an Android device using [Google's ML Kit](https://developers.google.com/ml-kit) (which performs OCR on-device).  
  
The data is stored either on a local filesystem or on Backblaze B2. 
In the case of Backblaze B2, the data is E2E encrypted and the key is stored locally on your device.  
  
The extracted text and metadata is stored locally in OpenSearch.

## Status

> [!NOTE]  
> This project is currently a work in progress but functions reasonably well for personal use cases.

## Getting Started

### Prerequisites

- Go
- [ocr-server](https://github.com/denysvitali/ocr-server) running somewhere reachable by the backend
- Docker (for running OpenSearch)
- Backblaze B2 / Filesystem
- An Apple AirScan (eSCL) compatible scanner, reachable by the backend
- [Zefix Tools](https://github.com/denysvitali/zefix-tools) for matching Swiss companies


### Starting ODI

```bash
# Clone the repository
git clone https://github.com/denysvitali/odi-backend.git
cd odi-backend
```

#### Configuring ODI

Using this `.env` file as an example, create a `.env` file in the root of the repository:

```bash
SCANNER_NAME=CanonScanner.local
OCR_API_ADDR=https://your-ocr-server.local:8443
OPENSEARCH_ADDR=https://localhost:9200
OPENSEARCH_USERNAME=admin
OPENSEARCH_PASSWORD=admin
OPENSEARCH_SKIP_TLS=true
ZEFIX_DSN=postgres://postgres:postgres@127.0.0.1:5432/postgres
LOG_LEVEL=debug


STORAGE_TYPE=b2 # or filesystem
B2_ACCOUNT=keychain:b2-account # or plaintext
B2_KEY=keychain:b2-key # or plaintext
B2_BUCKET_NAME=my-bucket
B2_PASSPHRASE=keychain:b2-passphrase # or plaintext
```

#### Running ODI

> [!IMPORTANT]  
> Make sure to start Zefix Tools's Postgres and make sure you've imported the data as explained in the [README](https://github.com/denysvitali/zefix-tools).
> As of today, Zefix tool is a prerequisite of ODI - in the future we plan to decouple them more.

##### OpenSearch

```bash
docker-compose up -d
```

##### Backend

```bash
source .env
go run ./cmd/odi-backend
```

##### Indexing

Start indexing your first documents by running the following command:

```bash
go run ./cmd/ingestor
```

Ingestor will contact your AirScan scanner and start processing documents. For each document, it will:
- Scan the document
- OCR the document
- Extract dates, companies and barcodes from the document
- Index the document text and metadata in OpenSearch
- Store the file (encrypted if blob storage) to your storage backend



## Security

> [!WARNING]  
> The current E2E encryption implemented for the blob storage backend is currently based on a single key. This means that
> only a certain amount of data can be encrypted before the key needs to be rotated.

> [!WARNING]  
> The code has not been audited. Use at your own risk.
> If you want a more robust solution, use the filesystem storage backend and provide a path to a FUSE encrypted filesystem.

If you discover a security issue, please report it to the email shown on [denv.it](https://denv.it).
