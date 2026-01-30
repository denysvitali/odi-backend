package storage

import (
	"fmt"

	"github.com/denysvitali/odi-backend/pkg/storage/b2"
	"github.com/denysvitali/odi-backend/pkg/storage/fs"
	"github.com/denysvitali/odi-backend/pkg/storage/model"
)

func SetupFsStorage(fsPath string) (model.RWStorage, error) {
	selectedStorage, err := fs.New(fsPath)
	if err != nil {
		return nil, fmt.Errorf("unable to create fs storage: %w", err)
	}
	return selectedStorage, nil
}

func SetupB2Storage(config b2.Config) (model.RWStorage, error) {
	selectedStorage, err := b2.New(config)
	if err != nil {
		return nil, fmt.Errorf("unable to create b2 storage: %w", err)
	}
	return selectedStorage, nil
}
