package ingestor_test

import (
	"bytes"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/h2non/gock"
	"github.com/sirupsen/logrus"

	"github.com/denysvitali/odi-backend/pkg/ingestor"
	"github.com/denysvitali/odi-backend/pkg/ocrclient"
	"github.com/denysvitali/odi-backend/pkg/storage/fs"
)

func TestMain(m *testing.M) {
	logrus.StandardLogger().SetLevel(logrus.DebugLevel)
	m.Run()
}

func TestIngest(t *testing.T) {
	scanner := os.Getenv("SCANNER_NAME")
	if scanner == "" {
		t.Skip("SCANNER_NAME not set, skipping test")
	}
	i := getIngestor(t)
	err := i.Ping()
	if err != nil {
		t.Fatal(err)
	}
	err = i.Ingest(scanner, "adf")
	if err != nil {
		t.Fatal(err)
	}
}

func getIngestor(t *testing.T) *ingestor.Ingestor {
	s, err := fs.New("/tmp/odi-backend")
	if err != nil {
		t.Fatal(err)
	}
	i, err := ingestor.New(ingestor.Config{
		OcrApiAddr:         "https://ocr-api.lan:8443",
		OpenSearchAddr:     "https://127.0.0.1:9200",
		OpenSearchUsername: "admin",
		OpenSearchPassword: "admin",
		OpenSearchSkipTLS:  true,
		ZefixDsn:           "postgres://postgres:postgres@localhost:5435/postgres",
		Storage:            s,
	})
	if err != nil {
		t.Fatal(err)
	}
	return i
}

type testScanner struct {
	files []io.Reader
	idx   int
}

func (t *testScanner) ScanPage() bool {
	if t.idx+1 <= len(t.files) {
		t.idx++
		return true
	}
	return false
}

func (t *testScanner) CurrentPage() io.Reader {
	if t.idx == 0 {
		return bytes.NewBuffer(nil)
	}
	return t.files[t.idx-1]
}

func (t *testScanner) Err() error {
	return nil
}

var _ ingestor.DocumentsScanner = (*testScanner)(nil)

func TestIngestWithExamplePictures(t *testing.T) {
	s := testScanner{
		files: []io.Reader{
			mustOpen("../../resources/testdata/private/1.jpg"),
			mustOpen("../../resources/testdata/private/2.jpg"),
		},
	}

	ocrApi := gock.New("https://ocr-api.lan:8443")
	ocrApi.
		Persist().
		Get("/healthz").
		Reply(200).
		BodyString(`{}`)

	gock.New("https://ocr-api.lan:8443").
		Persist().
		Post("/api/v1/ocr").
		Reply(http.StatusOK).
		JSON(
			ocrclient.OcrResult{
				TextBlocks: []ocrclient.TextBlock{
					{
						Text: "Hello World",
						BoundingBox: ocrclient.BoundingBox{
							Top:    0,
							Bottom: 100,
							Left:   0,
							Right:  20,
						},
					},
				},
			},
		)

	i := getIngestor(t)
	err := i.Ping()
	if err != nil {
		t.Fatal(err)
	}

	err = i.ScanPages(&s)
	if err != nil {
		t.Fatal(err)
	}
}

func mustOpen(s string) io.Reader {
	f, err := os.Open(s)
	if err != nil {
		panic(err)
	}

	b := bytes.NewBuffer(nil)
	_, err = io.Copy(b, f)
	if err != nil {
		panic(err)
	}
	return b
}
