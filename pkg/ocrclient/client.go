package ocrclient

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"

	"github.com/sirupsen/logrus"
)

type Client struct {
	http     *http.Client
	endpoint *url.URL
	mutex    sync.Mutex
}

var logger = logrus.StandardLogger().WithField("package", "ocr_client")

func (c *Client) Process(f io.Reader) (*OcrResult, error) {
	// We need to lock the mutex because the OCR API is not that fast
	c.mutex.Lock()
	defer c.mutex.Unlock()
	ocrUrl, err := c.endpoint.Parse("/api/v1/ocr")
	if err != nil {
		return nil, fmt.Errorf("unable to parse URL: %v", err)
	}
	req, err := http.NewRequest(http.MethodPost, ocrUrl.String(), f)
	if err != nil {
		return nil, fmt.Errorf("unable to create request: %v", err)
	}
	req.Header.Set("Content-Type", "image/jpeg")

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to perform HTTP request: %v", err)
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %s", res.Status)
	}

	var ocrResult OcrResult
	dec := json.NewDecoder(res.Body)
	err = dec.Decode(&ocrResult)

	return &ocrResult, err
}

// Healthz checks if the OCR service is healthy and returns true if it is.
func (c *Client) Healthz() (bool, error) {
	healthEndpoint, err := c.endpoint.Parse("/healthz")
	if err != nil {
		return false, err
	}
	res, err := c.http.Get(healthEndpoint.String())
	if err != nil {
		return false, err
	}

	if res.StatusCode == http.StatusOK {
		return true, nil
	}
	return false, nil
}

func New(endpoint string) (*Client, error) {
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return nil, fmt.Errorf("scheme %s is not supported", u.Scheme)
	}

	return &Client{
		endpoint: u,
		http:     http.DefaultClient,
	}, nil
}

func (c *Client) SetHttpTransport(transport http.RoundTripper) {
	c.http.Transport = transport
}
