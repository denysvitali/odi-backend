package main

import (
	"os"
	"path"
	"sync"

	"github.com/denysvitali/odi-backend/pkg/cli"

	"github.com/alexflint/go-arg"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/denysvitali/odi-backend/pkg/indexer"
	"github.com/denysvitali/odi-backend/pkg/models"
)

type argsT struct {
	InputDir string `arg:"positional,required"`

	Debug                        *bool  `arg:"-D,--debug,env:OCR_CLIENT_DEBUG"`
	OcrApi                       string `arg:"-o,--ocr-api,env:OCR_API_ADDR,required" help:"Address of the OCR API"`
	OcrApiCaPath                 string `arg:"--ocr-api-ca-path,env:OCR_API_CA_PATH"`
	OpenSearchAddr               string `arg:"-a,--os-address,env:OPENSEARCH_ADDR,required"`
	OpenSearchInsecureSkipVerify bool   `arg:"--insecure,env:OPENSEARCH_INSECURE_SKIP_VERIFY"`
	OpenSearchPassword           string `arg:"-p,--os-password,env:OPENSEARCH_PASSWORD,required"`
	OpenSearchUsername           string `arg:"-u,--os-username,env:OPENSEARCH_USERNAME,required"`
	Workers                      int    `arg:"-w" default:"4"`
	ZefixDsn                     string `arg:"--zefix-dsn,env:ZEFIX_DSN,required" help:"DSN to connect to the Zefix database"`
}

var args argsT

var log = logrus.StandardLogger()

func main() {
	arg.MustParse(&args)
	if err := cli.FillKeychainValues(&args); err != nil {
		log.Fatalf("fill keychain values: %v", err)
	}
	run()
}

func run() {
	if args.Debug != nil && *args.Debug {
		log.SetLevel(logrus.DebugLevel)
	}

	if args.Workers <= 0 {
		args.Workers = 4
		log.Warnf("workers cannot be <= 0, resetting value to %d", args.Workers)
	}
	opts := []indexer.Option{
		indexer.WithOpenSearchUsername(args.OpenSearchUsername),
		indexer.WithOpenSearchPassword(args.OpenSearchPassword),
		indexer.WithOcrApiCAPath(args.OcrApiCaPath),
	}
	if args.OpenSearchInsecureSkipVerify {
		opts = append(opts, indexer.WithOpenSearchSkipTLS())
	}

	idx, err := indexer.New(
		args.OpenSearchAddr,
		args.OcrApi,
		args.ZefixDsn,
		opts...,
	)
	if err != nil {
		log.Fatalf("unable to create indexer: %v", err)
	}

	res, err := idx.PingOpensearch()
	if err != nil {
		log.Fatalf("unable to ping OpenSearch: %v", err)
	}

	if res.IsError() {
		log.Fatalf("failed ping: %s", res.Status())
	}

	workers := args.Workers
	ch := make(chan models.ScannedPage)
	wg := sync.WaitGroup{}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		w := indexer.NewWorker(i, ch)
		w.SetIndexer(idx)
		go w.Start(&wg)
	}

	scanId := uuid.NewString()
	seq := 0
	for _, file := range listFiles(args.InputDir) {
		if !file.IsDir() {
			seq++
			f, err := os.Open(path.Join(args.InputDir, file.Name()))
			if err != nil {
				log.Errorf("unable to open file: %v", err)
				continue
			}
			ch <- models.ScannedPage{
				Reader:     f,
				ScanId:     scanId,
				SequenceId: seq,
			}
		}
	}
	close(ch)
	wg.Wait()
	log.Infof("done")
}

func listFiles(dir string) []os.DirEntry {
	d, err := os.ReadDir(dir)
	if err != nil {
		log.Fatalf("unable to read directory: %v", err)
	}
	return d
}
