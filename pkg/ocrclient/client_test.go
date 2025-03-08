package ocrclient_test

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/denysvitali/odi-backend/pkg/ocrclient"
	"github.com/denysvitali/odi-backend/pkg/ocrclient/caroundtripper"
)

func getFile(t *testing.T, path string) *os.File {
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("unable to open file: %v", err)
	}
	return f
}

func getClient(t *testing.T) *ocrclient.Client {
	ocrApiAddr := os.Getenv("OCR_API_ADDR")
	if ocrApiAddr == "" {
		t.Skip("OCR_API_ADDR not set, skipping test")
	}
	c, err := ocrclient.New(os.Getenv("OCR_API_ADDR"))
	if err != nil {
		t.Fatalf("unable to create client: %v", err)
	}

	caRoundtripper, err := caroundtripper.New(os.Getenv("OCR_API_CA_PATH"))
	if err != nil {
		t.Fatalf("unable to create CA client: %v", err)
	}

	c.SetHttpTransport(caRoundtripper)
	return c
}

func TestClient(t *testing.T) {
	c := getClient(t)
	healthy, err := c.Healthz()
	assert.True(t, healthy)
	assert.Nil(t, err)

	f := getFile(t, "../../resources/receipt-1.jpg")
	defer f.Close()
	ocrResult, err := c.Process(f)
	if err != nil {
		t.Fatalf("unable to perform OCR: %v", err)
	}

	fmt.Printf("OCR Result: %v", ocrResult)
}

func TestClientPrivate(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	c := getClient(t)
	healthy, err := c.Healthz()
	assert.True(t, healthy)
	assert.Nil(t, err)

	f, err := os.CreateTemp(os.TempDir(), "*.json")
	if err != nil {
		t.Fatalf("unable to create temporary file: %v", err)
	}

	inputFile := getFile(t, "../../resources/receipt-1.jpg")
	defer inputFile.Close()
	ocrResult, err := c.Process(inputFile)
	if err != nil {
		t.Fatalf("unable to perform OCR: %v", err)
	}

	enc := json.NewEncoder(f)
	err = enc.Encode(ocrResult)
	if err != nil {
		t.Fatalf("unable to encode JSON: %v", err)
	}
	defer f.Close()
	fmt.Printf("output file: %s", f.Name())
	fmt.Printf("text = %s", ocrResult.Text())
}
