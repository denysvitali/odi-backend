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
	b2fs       fs.Fs
	bucketName string
	crypt      *odicrypt.OdiCrypt
}

func (b *B2) Store(page models.ScannedPage) (err error) {
	ctx := context.Background()

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

	obj, err := b.b2fs.Put(ctx, page.Reader, b.toStorageFile(page, fileSize), &fs.RangeOption{Start: 0, End: fileSize})
	if err != nil {
		return err
	}
	log.Debugf("obj=%+v", obj)
	return nil
}

func fileName(scanId string, sequenceNumber int) string {
	return fmt.Sprintf("%s/%d.jpg", scanId, sequenceNumber)
}

func (b *B2) toStorageFile(page models.ScannedPage, fileSize int64) fs.ObjectInfo {
	return rclone.NewSourceFile(
		b.bucketName,
		fileName(page.ScanId, page.SequenceId),
		page.ScanTime,
		fileSize,
	)
}

func (b *B2) Retrieve(scanId string, sequenceId int) (*models.ScannedPage, error) {
	obj, err := b.b2fs.NewObject(context.Background(), fileName(scanId, sequenceId))
	if err != nil {
		if errors.Is(err, fs.ErrorObjectNotFound) {
			return nil, os.ErrNotExist
		}
		return nil, err
	}

	var reader io.ReadSeeker
	objReader, err := obj.Open(context.Background())
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
		ScanId:     scanId,
		SequenceId: sequenceId,
		ScanTime:   obj.ModTime(context.Background()),
	}, nil
}

// ListFiles returns a list of files for a given scan
func (b *B2) ListFiles(scanId string) ([]models.ScannedPage, error) {
	ctx := context.Background()
	objects, err := b.b2fs.List(ctx, scanId)
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
	scanId := path.Dir(obj.Remote())
	s.ScanId = scanId
	fileName = strings.TrimSuffix(fileName, ".jpg")
	seqId, err := strconv.ParseInt(fileName, 10, 64)
	if err == nil {
		s.SequenceId = int(seqId)
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

	b2fs, err := rcloneb2.NewFs(context.Background(),
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
		b2fs:       b2fs,
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
