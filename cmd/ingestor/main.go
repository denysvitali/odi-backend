package main

import (
	"strings"

	"github.com/alexflint/go-arg"
	"github.com/sirupsen/logrus"

	"github.com/denysvitali/odi-backend/pkg/cli"
	"github.com/denysvitali/odi-backend/pkg/ingestor"
	"github.com/denysvitali/odi-backend/pkg/logutils"
	"github.com/denysvitali/odi-backend/pkg/storage"
	"github.com/denysvitali/odi-backend/pkg/storage/b2"
	"github.com/denysvitali/odi-backend/pkg/storage/model"
)

var args struct {
	B2AccountId        string `arg:"--b2-account-id,env:B2_ACCOUNT" help:"Account for B2 storage - when using the b2 storage"`
	B2AccountKey       string `arg:"--b2-account-key,env:B2_KEY" help:"Key for B2 storage - when using the b2 storage"`
	B2BucketName       string `arg:"--b2-bucket-name,env:B2_BUCKET_NAME" help:"Bucket Name for B2 storage - when using the b2 storage"`
	B2Passphrase       string `arg:"--b2-passphrase,env:B2_PASSPHRASE" help:"Passphrase for B2 storage (optional) - when using the b2 storage"`
	FsPath             string `arg:"--fs-path,env:FS_PATH" help:"Path to the directory where to store the files - when using the fs storage"`
	LogLevel           string `arg:"--log-level,env:LOG_LEVEL" default:"info"`
	OcrApiAddr         string `arg:"--ocr-api-addr,required,env:OCR_API_ADDR"`
	OpenSearchAddr     string `arg:"--opensearch-addr,required,env:OPENSEARCH_ADDR"`
	OpenSearchPassword string `arg:"--opensearch-password,env:OPENSEARCH_PASSWORD"`
	OpenSearchSkipTLS  bool   `arg:"--opensearch-skip-tls,env:OPENSEARCH_SKIP_TLS"`
	OpenSearchUsername string `arg:"--opensearch-username,env:OPENSEARCH_USERNAME"`
	ScannerName        string `arg:"--scanner-name,env:SCANNER_NAME,required"`
	Source             string `arg:"--source,env:SOURCE" help:"Feeder or Platen" default:"Feeder"`
	StorageType        string `arg:"--storage-type,env:STORAGE_TYPE,required" help:"Type of storage to use"`
	ZefixDsn           string `arg:"--zefix-dsn,env:ZEFIX_DSN,required" help:"DSN to connect to the Zefix database"`
}

var log = logrus.StandardLogger()

func main() {
	arg.MustParse(&args)
	logutils.SetLoggerLevel(args.LogLevel)

	if err := cli.FillKeychainValues(&args); err != nil {
		log.Fatalf("unable to fill keychain values: %v", err)
	}

	log.Debugf("getting storage")
	selectedStorage := getStorage()
	log.Debugf("creating ingestor")
	i, err := ingestor.New(ingestor.Config{
		OcrApiAddr:         args.OcrApiAddr,
		OpenSearchAddr:     args.OpenSearchAddr,
		OpenSearchPassword: args.OpenSearchPassword,
		OpenSearchSkipTLS:  args.OpenSearchSkipTLS,
		OpenSearchUsername: args.OpenSearchUsername,
		Storage:            selectedStorage,
		ZefixDsn:           args.ZefixDsn,
	})
	if err != nil {
		log.Fatalf("unable to create ingestor: %v", err)
	}
	log.Debugf("starting to ingest")
	err = i.Ingest(args.ScannerName, args.Source)
	if err != nil {
		log.Fatalf("unable to ingest: %v", err)
	}
}

func getStorage() model.Storer {
	switch strings.ToLower(args.StorageType) {
	case "b2":
		return storage.SetupB2Storage(b2.Config{
			Account:    args.B2AccountId,
			BucketName: args.B2BucketName,
			Key:        args.B2AccountKey,
			Passphrase: args.B2Passphrase,
		})
	case "fs":
		return storage.SetupFsStorage(args.FsPath)
	}

	log.Fatalf("unknown storage type: %s", args.StorageType)
	return nil
}
