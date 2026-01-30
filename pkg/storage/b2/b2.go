package b2

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	rcloneb2 "github.com/rclone/rclone/backend/b2"
	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/config/configmap"
	"github.com/sirupsen/logrus"
	"io"
	"os"
	"path"
	"strconv"
	"strings"

	odicrypt "github.com/denysvitali/odi-backend/pkg/crypt"
	"github.com/denysvitali/odi-backend/pkg/models"
	"github.com/denysvitali/odi-backend/pkg/storage/model"
	"github.com/denysvitali/odi-backend/pkg/storage/rclone"
)

var log = logrus.StandardLogger().WithField("package", "storage/b2")
var _ model.Storer = (*B2)(nil)
var _ model.Retriever = (*B2)(nil)

type B2 struct {
	b2FS       fs.Fs
	bucketName string
	crypt      *odicrypt.OdiCrypt
}

func (b *B2) Store(ctx context.Context, page models.ScannedPage) (err error) {

	if b.crypt != nil {
		// The nonce needs to be unique, but not secure.
		// It should not be reused for more than 64GB of data for the same key.
		page.Reader, err = b.crypt.Encrypt(page.Reader)
		if err != nil {
			return err
		}
	}

	fileSize, err := page.Reader.Seek(0, io.SeekEnd)
	if err != nil {
		return err
	}
	_, err = page.Reader.Seek(0, io.SeekStart)
	if err != nil {
		return err
	}

	obj, err := b.b2FS.Put(ctx, page.Reader, b.toStorageFile(page, fileSize), &fs.RangeOption{Start: 0, End: fileSize})
	if err != nil {
		return err
	}
	log.Debugf("obj=%+v", obj)
	return nil
}

func fileName(scanID string, sequenceNumber int) string {
	return fmt.Sprintf("%s/%d.jpg", scanID, sequenceNumber)
}

func (b *B2) toStorageFile(page models.ScannedPage, fileSize int64) fs.ObjectInfo {
	return rclone.NewSourceFile(
		b.bucketName,
		fileName(page.ScanID, page.SequenceID),
		page.ScanTime,
		fileSize,
	)
}

func (b *B2) Retrieve(ctx context.Context, scanID string, sequenceId int) (*models.ScannedPage, error) {
	obj, err := b.b2FS.NewObject(ctx, fileName(scanID, sequenceId))
	if err != nil {
		if errors.Is(err, fs.ErrorObjectNotFound) {
			return nil, os.ErrNotExist
		}
		return nil, err
	}

	var reader io.ReadSeeker
	objReader, err := obj.Open(ctx)
	if err != nil {
		return nil, err
	}

	if b.crypt != nil {
		reader, err = b.crypt.Decrypt(objReader)
		if err != nil {
			return nil, err
		}
	} else {
		buffer := bytes.NewBuffer(nil)
		_, err = io.Copy(buffer, objReader)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(buffer.Bytes())
	}

	return &models.ScannedPage{
		Reader:     reader,
		ScanID:     scanID,
		SequenceID: sequenceId,
		ScanTime:   obj.ModTime(ctx),
	}, nil
}

// ListFiles returns a list of files for a given scan
func (b *B2) ListFiles(scanID string) ([]models.ScannedPage, error) {
	ctx := context.Background()
	objects, err := b.b2FS.List(ctx, scanID)
	if err != nil {
		return nil, err
	}

	var files []models.ScannedPage
	for _, obj := range objects {
		files = append(files, objToScannedPage(obj))
	}
	return files, nil
}

func objToScannedPage(obj fs.DirEntry) models.ScannedPage {
	s := models.ScannedPage{}
	fileName := path.Base(obj.Remote())
	scanID := path.Dir(obj.Remote())
	s.ScanID = scanID
	fileName = strings.TrimSuffix(fileName, ".jpg")
	seqId, err := strconv.ParseInt(fileName, 10, 64)
	if err == nil {
		s.SequenceID = int(seqId)
	}
	return s
}

type Config struct {
	Account    string
	Key        string
	BucketName string

	// Encryption specific
	Passphrase string
}

func New(config Config) (*B2, error) {
	if config.Account == "" {
		return nil, fmt.Errorf("account is required")
	}
	if config.Key == "" {
		return nil, fmt.Errorf("key is required")
	}
	if config.BucketName == "" {
		return nil, fmt.Errorf("bucket name is required")
	}

	if len(config.Passphrase) == 0 {
		log.Warnf("no passphrase provided, encryption will be disabled")
	}

	b2FS, err := rcloneb2.NewFs(context.Background(),
		"b2",
		config.BucketName+"/",
		configmap.Simple{
			"account":    config.Account,
			"key":        config.Key,
			"chunk_size": "5M",
		},
	)

	if err != nil {
		return nil, err
	}

	b := &B2{
		bucketName: config.BucketName,
		b2FS:       b2FS,
	}

	if len(config.Passphrase) != 0 {
		// Get key from passphrase
		b.crypt, err = odicrypt.New(config.Passphrase)
		if err != nil {
			return nil, err
		}
	}

	return b, nil
}
