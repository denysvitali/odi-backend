package zefix_test

import (
	"crypto/tls"
	"net/http"
	"os"
	"testing"

	"github.com/opensearch-project/opensearch-go/v2"

	"github.com/denysvitali/odi-backend/pkg/zefix"
)

func getClient(t *testing.T) *zefix.Processor {
	p, err := zefix.New("postgres://postgres:postgres@localhost:5435/postgres")
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func getOpenSearchClient(t *testing.T) *opensearch.Client {
	opensearchAddr := os.Getenv("OPENSEARCH_ADDR")
	if opensearchAddr == "" {
		t.Skip("OPENSEARCH_ADDR not set, skipping test")
	}
	cfg := opensearch.Config{
		Addresses: []string{opensearchAddr},
		Username:  os.Getenv("OPENSEARCH_USERNAME"),
		Password:  os.Getenv("OPENSEARCH_PASSWORD"),
	}
	if os.Getenv("OPENSEARCH_SKIP_TLS") == "true" {
		cfg.Transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	}
	osClient, err := opensearch.NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return osClient
}

func TestProcessor(t *testing.T) {
	p := getClient(t)
	osClient := getOpenSearchClient(t)
	err := p.ProcessFromOpenSearch(osClient, "documents")
	if err != nil {
		t.Fatal(err)
	}
}

func TestProcessor_FindCompanies(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	text := `Baloise Assicurazione SA
Aeschengraben 21, Casella postale
4002 Basel
www.baloise.ch
Servizio clientela 00800 24 800 800
servizioclientela@baloise.ch`

	p := getClient(t)
	companies := p.FindCompanies(text)
	if len(companies) != 1 {
		t.Fatalf("Expected 1 company, got %d", len(companies))
	}
}
func TestProcessor_FindCompanies2(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}
	text := `Konto / Zahlbar an
9403 Goldach
labor team w ag
Referenz
Empfangsschein
Zahlbar durch
Herr
Herr
zahlbar innert 30 Tagen
40
Eingang
Währung Betrag
CHF
Patient /lArzt
Annahmestelle
Zahlteil
Postfach, 9001 St. Gallen
Tel, +41 71 844 59 59
Fax +41 71 844 45 46
www.team-w.ch`

	p := getClient(t)
	companies := p.FindCompanies(text)
	if len(companies) != 1 {
		t.Fatalf("Expected 1 company, got %d", len(companies))
	}
}
