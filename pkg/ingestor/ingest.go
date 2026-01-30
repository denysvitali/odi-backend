package ingestor

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stapelberg/airscan"
	"github.com/stapelberg/airscan/preset"

	"github.com/denysvitali/odi-backend/pkg/indexer"
	"github.com/denysvitali/odi-backend/pkg/models"
	"github.com/denysvitali/odi-backend/pkg/storage/model"
)

var log = logrus.StandardLogger()

const (
	DefaultWorkers = 10
	// PageScanDelay is the delay between scanning pages to prevent overwhelming the scanner
	PageScanDelay = 100 * time.Millisecond
)

type Config struct {
	OcrAPIAddr         string
	OpenSearchAddr     string
	OpenSearchUsername string
	OpenSearchPassword string
	OpenSearchSkipTLS  bool
	ZefixDsn           string
	Storage            model.Storer
}

type Ingestor struct {
	idx     *indexer.Indexer
	storage model.Storer
}

func New(config Config) (*Ingestor, error) {
	var opts []indexer.Option
	if config.OpenSearchUsername != "" {
		opts = append(opts, indexer.WithOpenSearchUsername(config.OpenSearchUsername))
	}
	if config.OpenSearchPassword != "" {
		opts = append(opts, indexer.WithOpenSearchPassword(config.OpenSearchPassword))
	}
	if config.OpenSearchSkipTLS {
		opts = append(opts, indexer.WithOpenSearchSkipTLS())
	}
	idx, err := indexer.New(
		config.OpenSearchAddr, config.OcrAPIAddr, config.ZefixDsn,
		opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create indexer: %w", err)
	}

	ing := &Ingestor{idx: idx, storage: config.Storage}

	// Check that everything works:
	log.Debugf("Pinging services")
	if err := ing.Ping(context.Background()); err != nil {
		return nil, fmt.Errorf("unable to ping services: %w", err)
	}
	return ing, err
}

func (i *Ingestor) ScanPages(ctx context.Context, scanner DocumentsScanner, workers int) error {
	if workers <= 0 {
		workers = DefaultWorkers
	}

	pageChan := make(chan models.ScannedPage, workers)
	wg := sync.WaitGroup{}

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go i.processPage(ctx, pageChan, &wg)
	}

	scanID := uuid.NewString()
	seq := 0
	for scanner.ScanPage() {
		seq++
		b, err := io.ReadAll(scanner.CurrentPage())
		if err != nil {
			return fmt.Errorf("unable to read page: %w", err)
		}
		pageChan <- models.ScannedPage{
			Reader:     bytes.NewReader(b),
			ScanID:     scanID,
			SequenceID: seq,
			ScanTime:   time.Now(),
		}
		time.Sleep(PageScanDelay) // Slow down infinite loops
	}
	close(pageChan)
	wg.Wait()
	return nil
}

func (i *Ingestor) ScanPagesWithDefaultWorkers(ctx context.Context, scanner DocumentsScanner) error {
	return i.ScanPages(ctx, scanner, DefaultWorkers)
}

// Ingest takes care of connecting to the specified scanner, processes the document via OCR and outputs that to OpenSearch
func (i *Ingestor) Ingest(ctx context.Context, scannerName string, source string) error {
	c := airscan.NewClient(scannerName)
	settings := preset.GrayscaleA4ADF()
	settings.Duplex = false
	settings.ColorMode = "RGB24"
	settings.DocumentFormat = "image/jpeg"
	settings.InputSource = source

	job, err := c.Scan(settings)
	if err != nil {
		return fmt.Errorf("unable to create scan job: %w", err)
	}
	err = i.ScanPages(ctx, job, DefaultWorkers)
	return err
}

func (i *Ingestor) processPage(ctx context.Context, pageChan <-chan models.ScannedPage, wg *sync.WaitGroup) {
	defer wg.Done()
	for page := range pageChan {
		i.processPageInner(ctx, page)
	}
}

func (i *Ingestor) processPageInner(ctx context.Context, page models.ScannedPage) {

	// Read all page data into a single buffer to avoid multiple copies
	pageData, err := io.ReadAll(page.Reader)
	if err != nil {
		log.Errorf("unable to read page: %v", err)
		return
	}

	// Store the page using the same byte slice (avoids double buffering)
	err = i.storage.Store(ctx, models.ScannedPage{
		Reader:     bytes.NewReader(pageData),
		ScanID:     page.ScanID,
		SequenceID: page.SequenceID,
	})
	if err != nil {
		log.Errorf("unable to store page: %v", err)
		return
	}

	// Reuse the same byte slice for OCR/indexing
	page.Reader = bytes.NewReader(pageData)
	i.ocrAndIndex(ctx, page)
}

func (i *Ingestor) ocrAndIndex(ctx context.Context, page models.ScannedPage) {
	log.Debugf("ingesting page %d of scan %q", page.SequenceID, page.ScanID)
	err := i.idx.Index(ctx, page)
	if err != nil {
		log.Errorf("unable to index: %v", err)
	}
}

// Ping makes sure the two APIs (OCR and OpenSearch) are reachable
func (i *Ingestor) Ping(ctx context.Context) error {
	log.Debugf("Pinging OpenSearch")
	res, err := i.idx.PingOpensearch(ctx)
	if err != nil {
		return fmt.Errorf("unable to ping OpenSearch: %w", err)
	}
	if res.IsError() {
		return fmt.Errorf("unable to ping OpenSearch: %s", res.Status())
	}

	// Ping OCR
	log.Debugf("Pinging OCR API")
	h, err := i.idx.PingOcrApi()
	if err != nil {
		return fmt.Errorf("unable to ping OCR API: %w", err)
	}
	if !h {
		return fmt.Errorf("OCR API is not healthy")
	}

	log.Debugf("Pinging Zefix")
	err = i.idx.PingZefix()
	return nil
}
