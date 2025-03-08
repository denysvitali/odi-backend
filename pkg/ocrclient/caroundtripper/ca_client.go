package caroundtripper

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io"
	"net/http"
	"os"
)

var _ http.RoundTripper = (*Client)(nil)

type Client struct {
	transport *http.Transport
}

func (c Client) RoundTrip(request *http.Request) (*http.Response, error) {
	return c.transport.RoundTrip(request)
}

// New creates a http.Client that only trusts the CA
// specified in the caPath.
func New(caPath string) (*Client, error) {
	caFile, err := os.Open(caPath)
	if err != nil {
		return nil, err
	}

	caBytes, err := io.ReadAll(caFile)
	if err != nil {
		return nil, err
	}
	block, rest := pem.Decode(caBytes)

	if block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("invalid pem block type %s, expected CERTIFICATE", block.Type)
	}

	if len(rest) > 0 {
		return nil, fmt.Errorf("rest contaisn more than 0 bytes")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("unable to parse certificate: %v", err)
	}

	certPool := x509.NewCertPool()
	certPool.AddCert(cert)

	t := http.Transport{
		TLSClientConfig: &tls.Config{
			RootCAs: certPool,
		},
		ForceAttemptHTTP2: true,
	}

	return &Client{
		transport: &t,
	}, nil
}
