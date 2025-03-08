package models

import (
	"time"

	"github.com/denysvitali/zefix-tools/pkg/zefix"
)

type Document struct {
	Date               *time.Time      `json:"date,omitempty"`
	Text               string          `json:"text,omitempty"`
	Barcode            *Barcode        `json:"barcode,omitempty"`
	AdditionalBarcodes []Barcode       `json:"additionalBarcodes"`
	Company            *zefix.Company  `json:"company,omitempty"`
	Companies          []zefix.Company `json:"companies,omitempty"`
	Dates              []time.Time     `json:"dates,omitempty"`
	IndexedAt          time.Time       `json:"indexedAt,omitempty"`

	// Scan specific fields
	ScanId     string `json:"scanId"`
	SequenceId int    `json:"sequenceId"`
}
