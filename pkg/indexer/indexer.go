package indexer

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
	"github.com/sirupsen/logrus"

	"github.com/denysvitali/go-datesfinder"
	swissqrcode "github.com/denysvitali/go-swiss-qr-bill"

	"github.com/denysvitali/odi-backend/pkg/models"
	"github.com/denysvitali/odi-backend/pkg/ocrclient"
	"github.com/denysvitali/odi-backend/pkg/ocrclient/caroundtripper"
	"github.com/denysvitali/odi-backend/pkg/ocrtext"
	"github.com/denysvitali/odi-backend/pkg/zefix"
)

type Indexer struct {
	opensearchAddr               string
	opensearchUsername           string
	opensearchPassword           string
	opensearchInsecureSkipVerify bool
	documentsIndex               string
	ocrAPIAddr                   string
	ocrAPICaPath                 string
	zefixDsn                     string

	opensearchClient *opensearch.Client
	ocrClient        *ocrclient.Client
	zefixProcessor   *zefix.Processor

	initCalled         bool
	mergeDistance      float64
	horizontalDistance float64
}

const (
	DefaultDocumentsIndex = "documents"

	// DefaultMergeDistance is the default maximum vertical distance between text blocks to be merged
	DefaultMergeDistance = 150
	// DefaultHorizontalDistance is the default maximum horizontal distance between text blocks to be merged
	DefaultHorizontalDistance = 10
)

type Option func(*Indexer)

var log = logrus.StandardLogger().WithField("package", "indexer")

func New(opensearchAddr string, ocrAPIAddr string, zefixDsn string, opts ...Option) (*Indexer, error) {
	idx := &Indexer{
		opensearchAddr:     opensearchAddr,
		ocrAPIAddr:         ocrAPIAddr,
		zefixDsn:           zefixDsn,
		documentsIndex:     DefaultDocumentsIndex,
		mergeDistance:      DefaultMergeDistance,
		horizontalDistance: DefaultHorizontalDistance,
	}
	for _, opt := range opts {
		opt(idx)
	}
	if err := idx.init(); err != nil {
		return nil, fmt.Errorf("init indexer: %w", err)
	}
	return idx, nil
}

func (i *Indexer) PingOcrApi() (bool, error) {
	err := i.ensureOcrApiClient()
	if err != nil {
		return false, fmt.Errorf("ensure OCR API client: %w", err)
	}

	return i.ocrClient.Healthz()
}

func (i *Indexer) PingOpensearch(ctx context.Context) (*opensearchapi.Response, error) {
	err := i.ensureOpensearchClient()
	if err != nil {
		return nil, fmt.Errorf("ensure opensearch client: %w", err)
	}

	req := opensearchapi.PingRequest{}
	return req.Do(ctx, i.opensearchClient)
}

func (i *Indexer) ensureOpensearchClient() error {
	if i.opensearchClient != nil {
		return nil
	}

	var err error
	i.opensearchClient, err = opensearch.NewClient(opensearch.Config{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: i.opensearchInsecureSkipVerify},
		},
		Addresses: []string{i.opensearchAddr},
		Username:  i.opensearchUsername,
		Password:  i.opensearchPassword,
	})
	if err != nil {
		return fmt.Errorf("create opensearch client: %w", err)
	}
	return nil
}

func (i *Indexer) ensureOcrApiClient() error {
	if i.ocrClient != nil {
		return nil
	}

	var err error
	i.ocrClient, err = ocrclient.New(i.ocrAPIAddr)
	if err != nil {
		return fmt.Errorf("create OCR client: %w", err)
	}

	if i.ocrAPICaPath != "" {
		caRoundTripper, err := caroundtripper.New(i.ocrAPICaPath)
		if err != nil {
			return fmt.Errorf("create CA round tripper: %w", err)
		}
		i.ocrClient.SetHTTPTransport(caRoundTripper)
	}
	return nil
}

func (i *Indexer) init() error {
	err := i.ensureOcrApiClient()
	if err != nil {
		return fmt.Errorf("ocr client: %w", err)
	}
	err = i.ensureOpensearchClient()
	if err != nil {
		return fmt.Errorf("opensearchClient: %w", err)
	}

	err = i.ensureZefixClient()
	if err != nil {
		return fmt.Errorf("zefix client: %w", err)
	}

	// Create OpenSearch index
	err = i.createOpensearchIndex(context.Background())
	if err != nil {
		return fmt.Errorf("unable to create opensearch index: %w", err)
	}

	// Check if API ping works
	h, err := i.ocrClient.Healthz()
	if err != nil {
		return fmt.Errorf("unable to ping OCR API: %w", err)
	}

	if !h {
		return fmt.Errorf("OCR API is not healthy")
	}

	i.initCalled = true
	return nil
}

func (i *Indexer) ensureInitCalled() error {
	if !i.initCalled {
		return fmt.Errorf("init wasn't called")
	}
	return nil
}

func (i *Indexer) Index(ctx context.Context, page models.ScannedPage) error {
	log.Debugf("indexing %s", page.ID())
	err := i.ensureInitCalled()
	if err != nil {
		return fmt.Errorf("ensure init called: %w", err)
	}

	log.Debugf("processing %s via OCR client", page.ID())
	ocrResult, err := i.ocrClient.Process(page.Reader)
	if err != nil {
		return fmt.Errorf("ocr client failed: %w", err)
	}

	log.Debugf("getting text")
	documentText := i.getText(ocrResult)
	log.Debugf("zefixProcessor finds the companies")
	zefixCompanies := i.zefixProcessor.FindCompanies(documentText)
	log.Debugf("found %d companies", len(zefixCompanies))

	jsonBuffer := bytes.NewBuffer(nil)
	enc := json.NewEncoder(jsonBuffer)
	log.Debugf("getting barcodes for %s", page.ID())
	barcodes := i.getBarcodes(ocrResult)
	var barcode *models.Barcode
	var additionalBarcodes []models.Barcode
	if len(barcodes) > 1 {
		additionalBarcodes = barcodes[1:]
	}

	if len(barcodes) >= 1 {
		barcode = &barcodes[0]
	}
	dates := getDocumentDates(ocrResult)
	d := &models.Document{
		Text:               documentText,
		Barcode:            barcode,
		AdditionalBarcodes: additionalBarcodes,
		IndexedAt:          time.Now(),
		ScanID:             page.ScanID,
		SequenceID:         page.SequenceID,
	}
	if len(dates) > 0 {
		d.Date = &dates[0]
		d.Dates = dates
	}
	if len(zefixCompanies) > 0 {
		log.Debugf("found %d companies", len(zefixCompanies))
		d.Company = &zefixCompanies[0]
		d.Companies = zefixCompanies
	}
	err = enc.Encode(d)
	if err != nil {
		return fmt.Errorf("unable to encode JSON: %w", err)
	}

	log.Debugf("indexing %s", page.ID())

	req := opensearchapi.IndexRequest{
		Index:      i.documentsIndex,
		DocumentID: page.ID(),
		Body:       jsonBuffer,
		OpType:     "index",
	}
	res, err := req.Do(ctx, i.opensearchClient)
	if err != nil {
		return fmt.Errorf("index document: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		errorMessage := decodeError(res.Body)
		return fmt.Errorf("opensearch returned an invalid status %s: %s", res.Status(), errorMessage)
	}
	log.Debugf("indexed %s", page.ID())
	return nil
}

func decodeError(body io.ReadCloser) string {
	var errorMessage struct {
		Error string `json:"error"`
	}
	dec := json.NewDecoder(body)
	if err := dec.Decode(&errorMessage); err != nil {
		return fmt.Sprintf("failed to decode error: %v", err)
	}
	return errorMessage.Error
}

// Given the result of the OCR, return the most likely date of the document
func getDocumentDates(result *ocrclient.OcrResult) []time.Time {
	// Try to parse the date from the text
	var dates []time.Time
	for _, t := range result.TextBlocks {
		d, err := datesfinder.FindDates(t.Text)
		if err != nil {
			continue
		}
		dates = append(dates, d...)
	}

	if len(dates) == 0 {
		return nil
	}
	return dates
}

func (i *Indexer) getBarcodes(result *ocrclient.OcrResult) []models.Barcode {
	if result == nil {
		return nil
	}

	var barcodes []models.Barcode
	for _, b := range result.Barcodes {
		if strings.HasPrefix(b.RawValue, "SPC") {
			// Try to parse Swiss QR Bill
			qrCode, err := swissqrcode.Decode(b.RawValue)
			if err != nil {
				log.Warnf("unable to decode Swiss QR Bill: %v", err)
				barcodes = append(barcodes, models.Barcode{Text: b.RawValue})
				continue
			}
			barcodes = append(barcodes, models.Barcode{QRBill: qrCode})
		} else {
			barcodes = append(barcodes, models.Barcode{Text: b.RawValue})
		}
	}
	return barcodes
}

func (i *Indexer) getText(result *ocrclient.OcrResult) string {
	return ocrtext.GetText(result, i.mergeDistance, i.horizontalDistance)
}

func (i *Indexer) createOpensearchIndex(ctx context.Context) error {
	req := opensearchapi.IndicesCreateRequest{Index: i.documentsIndex}
	res, err := req.Do(ctx, i.opensearchClient)
	if err != nil {
		return fmt.Errorf("create index: %w", err)
	}
	if res.StatusCode == http.StatusBadRequest {
		// Index already exists
		return nil
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %s", res.Status())
	}

	return nil
}

func (i *Indexer) ensureZefixClient() error {
	var err error
	i.zefixProcessor, err = zefix.New(i.zefixDsn)
	if err != nil {
		return fmt.Errorf("create zefix client: %w", err)
	}
	return nil
}

func (i *Indexer) PingZefix() error {
	return i.zefixProcessor.Ping()
}
