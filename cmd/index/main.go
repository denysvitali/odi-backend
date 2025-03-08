package main

// This tool is used to index the files in the B2 bucket to apply new indexing rules
// or simply to re-index the files that failed to be indexed the first time.

import (
	"github.com/alexflint/go-arg"
	"github.com/sirupsen/logrus"

	"github.com/denysvitali/odi-backend/pkg/cli"

	"github.com/denysvitali/odi-backend/pkg/indexer"
	logutils "github.com/denysvitali/odi-backend/pkg/logutils"
	"github.com/denysvitali/odi-backend/pkg/storage/b2"
)

var args struct {
	ScanId string `arg:"positional,required"`

	B2Account          string `arg:"env:B2_ACCOUNT"`
	B2BucketName       string `arg:"env:B2_BUCKET_NAME"`
	B2Key              string `arg:"env:B2_KEY"`
	B2Passphrase       string `arg:"env:B2_PASSPHRASE"`
	LogLevel           string `arg:"--log-level,env:LOG_LEVEL" default:"info"`
	OcrApiAddr         string `arg:"--ocr-api-addr,required,env:OCR_API_ADDR"`
	OpenSearchAddr     string `arg:"--opensearch-addr,required,env:OPENSEARCH_ADDR"`
	OpenSearchPassword string `arg:"--opensearch-password,env:OPENSEARCH_PASSWORD"`
	OpenSearchSkipTLS  bool   `arg:"--opensearch-skip-tls,env:OPENSEARCH_SKIP_TLS"`
	OpenSearchUsername string `arg:"--opensearch-username,env:OPENSEARCH_USERNAME"`
	ZefixDsn           string `arg:"--zefix-dsn,env:ZEFIX_DSN,required" help:"DSN to connect to the Zefix database"`
}

var log = logrus.StandardLogger()

func main() {
	arg.MustParse(&args)
	if err := cli.FillKeychainValues(&args); err != nil {
		log.Fatalf("fill keychain values: %v", err)
	}
	logutils.SetLoggerLevel(args.LogLevel)
	b, err := b2.New(
		b2.Config{
			Account:    args.B2Account,
			Key:        args.B2Key,
			BucketName: args.B2BucketName,
			Passphrase: args.B2Passphrase,
		},
	)
	if err != nil {
		log.Fatalf("create b2 storage: %v", err)
	}

	var opts []indexer.Option
	if args.OpenSearchUsername != "" {
		opts = append(opts, indexer.WithOpenSearchUsername(args.OpenSearchUsername))
	}
	if args.OpenSearchPassword != "" {
		opts = append(opts, indexer.WithOpenSearchPassword(args.OpenSearchPassword))
	}
	if args.OpenSearchSkipTLS {
		opts = append(opts, indexer.WithOpenSearchSkipTLS())
	}
	if err != nil {
		log.Fatalf("create indexer: %v", err)
	}
	idx, err := indexer.New(
		args.OpenSearchAddr,
		args.OcrApiAddr,
		args.ZefixDsn,
		opts...,
	)
	if err != nil {
		log.Fatalf("create indexer: %v", err)
	}

	scanFiles, err := b.ListFiles(args.ScanId)
	if err != nil {
		log.Fatalf("list files: %v", err)
	}
	for _, f := range scanFiles {
		log.Infof("Indexing %s", f.Id())
		scannedPage, err := b.Retrieve(f.ScanId, f.SequenceId)
		if err != nil {
			log.Errorf("retrieve file %s: %v", f.Id(), err)
			continue
		}
		err = idx.Index(*scannedPage)
		if err != nil {
			log.Errorf("index file %s: %v", f.Id(), err)
			continue
		}
	}
}
