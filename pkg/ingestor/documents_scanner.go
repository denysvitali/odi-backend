package ingestor

import "io"

type DocumentsScanner interface {
	ScanPage() bool
	CurrentPage() io.Reader
	Err() error
}
