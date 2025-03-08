# documents-indexer

## Description

This is a simple tool that indexes documents in a 
directory by calling the [ocr-server](https://github.com/denysvitali/ocr-server)
app running on an Android device.

## Requirements

- Go
- Android device running [ocr-server](https://github.com/denysvitali/ocr-server)


## Compiling

```bash
go install github.com/denysvitali/documents-indexer/cmd/documents-indexer
```

## Usage

```bash
export OCR_API_ADDR=https://ocr-api.lan:8443
export OCR_API_CA_PATH=~/Documents/pki/root/certs/root.crt
export OPENSEARCH_ADDR=https://127.0.0.1:9200
export OPENSEARCH_INSECURE_SKIP_VERIFY=true
export OPENSEARCH_PASSWORD=admin
export OPENSEARCH_USERNAME=admin
```

```bash
documents-indexer ~/Documents/Scans
```