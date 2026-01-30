package model

import (
	"context"

	"github.com/denysvitali/odi-backend/pkg/models"
)

type Storer interface {
	Store(ctx context.Context, page models.ScannedPage) error
}

type Retriever interface {
	Retrieve(ctx context.Context, scanID string, sequenceNumber int) (*models.ScannedPage, error)
}

type RWStorage interface {
	Storer
	Retriever
}
