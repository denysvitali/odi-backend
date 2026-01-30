package fs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"

	"github.com/sirupsen/logrus"

	"github.com/denysvitali/odi-backend/pkg/models"
	"github.com/denysvitali/odi-backend/pkg/storage/model"
)

// validScanIDRegex matches only alphanumeric characters, hyphens, and underscores
var validScanIDRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// validateScanID validates that the scanID only contains safe characters
// and does not contain path traversal sequences
func validateScanID(scanID string) error {
	if scanID == "" {
		return fmt.Errorf("scanID cannot be empty")
	}
	if !validScanIDRegex.MatchString(scanID) {
		return fmt.Errorf("scanID contains invalid characters: only alphanumeric characters, hyphens, and underscores are allowed")
	}
	return nil
}

var log = logrus.StandardLogger().WithField("package", "storage/fs")

type Fs struct {
	dir string
}

func (fs *Fs) Retrieve(ctx context.Context, scanID string, sequenceNumber int) (*models.ScannedPage, error) {
	if err := validateScanID(scanID); err != nil {
		return nil, err
	}
	f, err := os.Open(path.Join(fs.dir, scanID, fmt.Sprintf("%d.jpg", sequenceNumber)))
	if err != nil {
		return nil, err
	}
	return &models.ScannedPage{
		ScanID:     scanID,
		SequenceID: sequenceNumber,
		Reader:     f,
	}, nil
}

func (fs *Fs) Store(ctx context.Context, page models.ScannedPage) error {
	if err := validateScanID(page.ScanID); err != nil {
		return err
	}
	// Check if directory exists
	_, err := os.Stat(path.Join(fs.dir, page.ScanID))
	if os.IsNotExist(err) {
		err = os.MkdirAll(path.Join(fs.dir, page.ScanID), 0755)
		if err != nil {
			return err
		}
	}

	f, err := os.Create(path.Join(fs.dir, page.ScanID, fmt.Sprintf("%d.jpg", page.SequenceID)))
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
			return nil, fmt.Errorf("unable to create storage directory: %w", err)
		}
	}

	fs := &Fs{
		dir: dir,
	}
	return fs, nil
}
