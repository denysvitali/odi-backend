package main

import (
	"strings"

	"github.com/denysvitali/odi-backend/pkg/cli"
	"github.com/denysvitali/odi-backend/pkg/storage/model"

	"github.com/alexflint/go-arg"

	backend "github.com/denysvitali/odi-backend"
	"github.com/denysvitali/odi-backend/pkg/logutils"
	"github.com/denysvitali/odi-backend/pkg/storage"
	"github.com/denysvitali/odi-backend/pkg/storage/b2"

	"github.com/sirupsen/logrus"
)

var args struct {
	B2AccountId          string `arg:"--b2-account-id,env:B2_ACCOUNT" help:"Account for B2 storage - when using the b2 storage"`
	B2AccountKey         string `arg:"--b2-account-key,env:B2_KEY" help:"Key for B2 storage - when using the b2 storage"`
	B2BucketName         string `arg:"--b2-bucket-name,env:B2_BUCKET_NAME" help:"Bucket Name for B2 storage - when using the b2 storage"`
	B2Passphrase         string `arg:"env:B2_PASSPHRASE" help:"Passphrase for B2 storage (optional) - when using the b2 storage"`
	FsPath               string `arg:"--fs-path,env:FS_PATH" help:"Path to the directory where to store the files - when using the fs storage"`
	ListenAddr           string `arg:"-L,--listen-addr" default:"127.0.0.1:8085"`
	LogLevel             string `arg:"--log-level,env:LOG_LEVEL" default:"info"`
	OsAddr               string `arg:"--opensearch-addr,required,env:OPENSEARCH_ADDR"`
	OsIndex              string `arg:"--opensearch-index,env:OPENSEARCH_INDEX" default:"documents"`
	OsInsecureSkipVerify bool   `arg:"--opensearch-insecure-skip-verify,env:OPENSEARCH_SKIP_TLS"`
	OsPassword           string `arg:"--opensearch-password,env:OPENSEARCH_PASSWORD"`
	OsUsername           string `arg:"--opensearch-username,env:OPENSEARCH_USERNAME"`
	StorageType          string `arg:"--storage-type,env:STORAGE_TYPE,required" help:"Type of storage to use"`
}

var log = logrus.StandardLogger()

func main() {
	arg.MustParse(&args)
	if err := cli.FillKeychainValues(&args); err != nil {
		log.Fatalf("fill keychain values: %v", err)
	}
	logutils.SetLoggerLevel(args.LogLevel)
	s, err := backend.New(
		args.OsAddr,
		args.OsUsername,
		args.OsPassword,
		args.OsInsecureSkipVerify,
		args.OsIndex,
		getStorage(),
	)
	if err != nil {
		log.Fatalf("create backend: %v", err)
	}

	err = s.Run(args.ListenAddr)
	if err != nil {
		log.Fatalf("listen: %v", err)
	}
}

func getStorage() model.RWStorage {
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
