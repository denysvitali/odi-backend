package ingestor

import (
	"bytes"
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

type Config struct {
	OcrApiAddr         string
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
		config.OpenSearchAddr, config.OcrApiAddr, config.ZefixDsn,
		opts...,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to create indexer: %w", err)
	}

	ing := &Ingestor{idx: idx, storage: config.Storage}

	// Check that everything works:
	log.Debugf("Pinging services")
	if err := ing.Ping(); err != nil {
		return nil, fmt.Errorf("unable to ping services: %w", err)
	}
	return ing, err
}

func (i *Ingestor) ScanPages(scanner DocumentsScanner) error {
	pageChan := make(chan models.ScannedPage)
	wg := sync.WaitGroup{}
	wg.Add(1)
	go i.processPage(pageChan, &wg)

	scanId := uuid.NewString()
	seq := 0
	for scanner.ScanPage() {
		seq++
		b, err := io.ReadAll(scanner.CurrentPage())
		if err != nil {
			return fmt.Errorf("unable to read page: %w", err)
		}
		pageChan <- models.ScannedPage{
			Reader:     bytes.NewReader(b),
			ScanId:     scanId,
			SequenceId: seq,
			ScanTime:   time.Now(),
		}
		time.Sleep(100 * time.Millisecond) // Slow down infinite loops
	}
	close(pageChan)
	wg.Wait()
	return nil
}

// Ingest takes care of connecting to the specified scanner, processes the document via OCR and outputs that to OpenSearch
func (i *Ingestor) Ingest(scannerName string, source string) error {
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
	err = i.ScanPages(job)
	return err
}

func (i *Ingestor) processPage(pageChan <-chan models.ScannedPage, wg *sync.WaitGroup) {
	for page := range pageChan {
		wg.Add(1)
		go i.processPageInner(page, wg)
	}
	wg.Done()
}

func (i *Ingestor) processPageInner(page models.ScannedPage, wg *sync.WaitGroup) {
	defer wg.Done()
	buffer := bytes.NewBuffer([]byte{})
	_, err := io.Copy(buffer, page.Reader)
	if err != nil {
		log.Errorf("unable to read page: %v", err)
		return
	}

	page.Reader = bytes.NewReader(buffer.Bytes())

	err = i.storage.Store(models.ScannedPage{
		Reader:     bytes.NewReader(buffer.Bytes()),
		ScanId:     page.ScanId,
		SequenceId: page.SequenceId,
	})
	if err != nil {
		log.Errorf("unable to store page: %v", err)
		return
	}

	i.ocrAndIndex(page)
}

func (i *Ingestor) ocrAndIndex(page models.ScannedPage) {
	log.Debugf("ingesting page %d of scan %q", page.SequenceId, page.ScanId)
	err := i.idx.Index(page)
	if err != nil {
		log.Errorf("unable to index: %v", err)
	}
}

// Ping makes sure the two APIs (OCR and OpenSearch) are reachable
func (i *Ingestor) Ping() error {
	log.Debugf("Pinging OpenSearch")
	res, err := i.idx.PingOpensearch()
	if err != nil {
		return fmt.Errorf("unable to ping OpenSearch: %v", err)
	}
	if res.IsError() {
		return fmt.Errorf("unable to ping OpenSearch: %v", res.Status())
	}

	// Ping OCR
	log.Debugf("Pinging OCR API")
	h, err := i.idx.PingOcrApi()
	if err != nil {
		return fmt.Errorf("unable to ping OCR API: %v", err)
	}
	if !h {
		return fmt.Errorf("OCR API is not healthy")
	}

	log.Debugf("Pinging Zefix")
	err = i.idx.PingZefix()
	return nil
}
