package fs

import (
	"fmt"
	"io"
	"os"
	"path"

	"github.com/sirupsen/logrus"

	"github.com/denysvitali/odi-backend/pkg/models"
	"github.com/denysvitali/odi-backend/pkg/storage/model"
)

var log = logrus.StandardLogger().WithField("package", "storage/fs")

type Fs struct {
	dir string
}

func (fs *Fs) Retrieve(scanId string, sequenceNumber int) (*models.ScannedPage, error) {
	f, err := os.Open(path.Join(fs.dir, scanId, fmt.Sprintf("%d.jpg", sequenceNumber)))
	if err != nil {
		return nil, err
	}
	return &models.ScannedPage{
		ScanId:     scanId,
		SequenceId: sequenceNumber,
		Reader:     f,
	}, nil
}

func (fs *Fs) Store(page models.ScannedPage) error {
	// Check if directory exists
	_, err := os.Stat(path.Join(fs.dir, page.ScanId))
	if os.IsNotExist(err) {
		err = os.MkdirAll(path.Join(fs.dir, page.ScanId), 0755)
		if err != nil {
			return err
		}
	}

	f, err := os.Create(path.Join(fs.dir, page.ScanId, fmt.Sprintf("%d.jpg", page.SequenceId)))
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(f, page.Reader); err != nil {
		return err
	}
	if _, err := page.Reader.Seek(0, io.SeekStart); err != nil {
		return err
	}
	log.Debugf("Created file %s", f.Name())
	return nil
}

var _ model.Storer = (*Fs)(nil)
var _ model.Retriever = (*Fs)(nil)

func New(dir string) (*Fs, error) {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			log.Fatalf("unable to create storage directory: %v", err)
		}
	}

	fs := &Fs{
		dir: dir,
	}
	return fs, nil
}
