package storage

import (
	"github.com/sirupsen/logrus"

	"github.com/denysvitali/odi-backend/pkg/storage/b2"
	"github.com/denysvitali/odi-backend/pkg/storage/fs"
	"github.com/denysvitali/odi-backend/pkg/storage/model"
)

var log = logrus.StandardLogger().WithField("package", "storage")

func SetupFsStorage(fsPath string) model.RWStorage {
	selectedStorage, err := fs.New(fsPath)
	if err != nil {
		log.Fatalf("unable to create fs storage: %v", err)
	}
	return selectedStorage
}

func SetupB2Storage(config b2.Config) model.RWStorage {
	selectedStorage, err := b2.New(config)
	if err != nil {
		log.Fatalf("unable to create b2 storage: %v", err)
	}
	return selectedStorage
}
