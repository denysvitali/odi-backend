package model

import "github.com/denysvitali/odi-backend/pkg/models"

type Storer interface {
	Store(models.ScannedPage) error
}

type Retriever interface {
	Retrieve(scanId string, sequenceNumber int) (*models.ScannedPage, error)
}

type RWStorage interface {
	Storer
	Retriever
}
