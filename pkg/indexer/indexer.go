package indexer

import (
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/opensearch-project/opensearch-go"
	"github.com/opensearch-project/opensearch-go/opensearchapi"
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
	ocrApiAddr                   string
	ocrApiCaPath                 string
	zefixDsn                     string

	opensearchClient *opensearch.Client
	ocrClient        *ocrclient.Client
	zefixProcessor   *zefix.Processor

	initCalled         bool
	mergeDistance      float64
	horizontalDistance float64
}

const DefaultDocumentsIndex = "documents"

type Option func(*Indexer)

var log = logrus.StandardLogger().WithField("package", "indexer")

func New(opensearchAddr string, ocrApiAddr string, zefixDsn string, opts ...Option) (*Indexer, error) {
	idx := &Indexer{
		opensearchAddr:     opensearchAddr,
		ocrApiAddr:         ocrApiAddr,
		zefixDsn:           zefixDsn,
		documentsIndex:     DefaultDocumentsIndex,
		mergeDistance:      150,
		horizontalDistance: 10,
	}
	for _, opt := range opts {
		opt(idx)
	}
	if err := idx.init(); err != nil {
		return nil, err
	}
	return idx, nil
}

func (i *Indexer) PingOcrApi() (bool, error) {
	err := i.ensureOcrApiClient()
	if err != nil {
		return false, err
	}

	return i.ocrClient.Healthz()
}

func (i *Indexer) PingOpensearch() (*opensearchapi.Response, error) {
	err := i.ensureOpensearchClient()
	if err != nil {
		return nil, err
	}

	req := opensearchapi.PingRequest{}
	return req.Do(context.Background(), i.opensearchClient)
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
	return err
}

func (i *Indexer) ensureOcrApiClient() error {
	if i.ocrClient != nil {
		return nil
	}

	var err error
	i.ocrClient, err = ocrclient.New(i.ocrApiAddr)
	if err != nil {
		return err
	}

	if i.ocrApiCaPath != "" {
		caRoundTripper, err := caroundtripper.New(i.ocrApiCaPath)
		if err != nil {
			return err
		}
		i.ocrClient.SetHttpTransport(caRoundTripper)
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
	err = i.createOpensearchIndex()
	if err != nil {
		return fmt.Errorf("unable to create opensearch index: %v", err)
	}

	// Check if API ping works
	h, err := i.ocrClient.Healthz()
	if err != nil {
		return fmt.Errorf("unable to ping OCR API: %v", err)
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

func (i *Indexer) Index(page models.ScannedPage) error {
	log.Debugf("indexing %s", page.Id())
	err := i.ensureInitCalled()
	if err != nil {
		return err
	}

	log.Debugf("processing %s via OCR client", page.Id())
	ocrResult, err := i.ocrClient.Process(page.Reader)
	if err != nil {
		return fmt.Errorf("ocr client failed: %v", err)
	}

	log.Debugf("getting text")
	documentText := i.getText(ocrResult)
	log.Debugf("zefixProcessor finds the companies")
	zefixCompanies := i.zefixProcessor.FindCompanies(documentText)
	log.Debugf("found %d companies", len(zefixCompanies))

	jsonBuffer := bytes.NewBuffer(nil)
	enc := json.NewEncoder(jsonBuffer)
	log.Debugf("getting barcodes for %s", page.Id())
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
		ScanId:             page.ScanId,
		SequenceId:         page.SequenceId,
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
		return fmt.Errorf("unable to encode JSON: %v", err)
	}

	log.Debugf("indexing %s", page.Id())

	req := opensearchapi.IndexRequest{
		Index:      i.documentsIndex,
		DocumentID: page.Id(),
		Body:       jsonBuffer,
		OpType:     "index",
	}
	res, err := req.Do(context.Background(), i.opensearchClient)
	if err != nil {
		return err
	}

	if res.StatusCode < 200 || res.StatusCode > 299 {
		errorMessage := decodeError(res.Body)
		return fmt.Errorf("opensearch returned an invalid status %s: %s", res.Status(), errorMessage)
	}
	log.Debugf("indexed %s", page.Id())
	return nil
}

func decodeError(body io.ReadCloser) string {
	var errorMessage struct {
		Error string `json:"error"`
	}
	dec := json.NewDecoder(body)
	dec.Decode(&errorMessage)
	return errorMessage.Error
}

// Given the result of the OCR, return the most likely date of the document
func getDocumentDates(result *ocrclient.OcrResult) []time.Time {
	// Try to parse the date from the text
	var dates []time.Time
	for _, t := range result.TextBlocks {
		d, _ := datesfinder.FindDates(t.Text)
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

func documentHash(reader io.Reader) (string, error) {
	h := sha1.New()
	_, err := io.Copy(h, reader)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (i *Indexer) createOpensearchIndex() error {
	req := opensearchapi.IndicesCreateRequest{Index: i.documentsIndex}
	res, err := req.Do(context.Background(), i.opensearchClient)
	if err != nil {
		return err
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
	return err
}

func (i *Indexer) PingZefix() error {
	return i.zefixProcessor.Ping()
}
