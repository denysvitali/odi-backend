package zefix

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
	"github.com/sirupsen/logrus"

	"github.com/denysvitali/go-datesfinder"

	"github.com/denysvitali/odi-backend/pkg/models"
	"github.com/denysvitali/zefix-tools/pkg/zefix"
)

var log = logrus.StandardLogger().WithField("package", "zefix")

type Processor struct {
	zefixClient *zefix.Client
}

func New(zefixDsn string) (*Processor, error) {
	zefixClient, err := zefix.New(zefixDsn)
	if err != nil {
		return nil, err
	}

	p := Processor{
		zefixClient: zefixClient,
	}
	return &p, nil
}

func (p *Processor) ProcessFromOpenSearch(osClient *opensearch.Client, index string) error {
	// Go through all the documents in the index and update them
	// by adding some new fields

	// 1. Get all the documents from the index
	size := 1000
	req := opensearchapi.SearchRequest{
		Index: []string{index},
		Sort:  []string{"date:asc"},
		Size:  &size,
	}

	res, err := req.Do(context.Background(), osClient)
	if err != nil {
		return err
	}

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", res.StatusCode)
	}

	// Parse response as JSON
	var result OpensearchResult[models.Document]
	dec := json.NewDecoder(res.Body)
	err = dec.Decode(&result)
	if err != nil {
		return err
	}

	// 2. Iterate over the documents
	for _, hit := range result.Hits.Hits {
		newSource := p.processText(hit.Source)
		newSourceBytes, err := json.Marshal(newSource)
		if err != nil {
			return err
		}

		// 3. Update the document
		req := opensearchapi.IndexRequest{
			Index:      index,
			DocumentID: hit.ID,
			Body:       strings.NewReader(string(newSourceBytes)),
		}

		res, err = req.Do(context.Background(), osClient)
		if err != nil {
			return err
		}

		if res.StatusCode != http.StatusOK {
			// Get the error from body
			body, err := io.ReadAll(res.Body)
			if err != nil {
				return err
			}
			return fmt.Errorf("unexpected status code: %d, %s", res.StatusCode, string(body))
		}
	}

	return nil
}

func (p *Processor) processText(document models.Document) models.Document {
	// Find dates
	dates, errors := datesfinder.FindDates(document.Text)
	printErrors(errors)
	log.Infof("found %d dates", len(dates))

	// Find companies
	companies := p.FindCompanies(document.Text)
	log.Infof("found %d companies", len(companies))

	if len(companies) > 0 {
		document.Company = &companies[0]
		document.Companies = companies
	}

	if len(dates) > 0 {
		document.Date = &dates[0]
		document.Dates = dates
	}

	return document
}

var companyRegexp = regexp.MustCompile("(?i)([A-zü() -]+) (?:AG|GmbH|SA|Sagl)")

func (p *Processor) FindCompanies(text string) []zefix.Company {
	companiesMap := make(map[string]zefix.Company)
	res := companyRegexp.FindAllStringSubmatch(text, -1)
	for _, company := range res {
		companyName := strings.TrimSpace(company[0])
		if _, ok := companiesMap[companyName]; ok {
			continue
		}
		if strings.HasPrefix(strings.ToLower("Post CH "), strings.ToLower(companyName)) {
			// Skip Post CH AG since it does appear on basically every document
			continue
		}
		log.Infof("found company: %s", companyName)
		c, err := p.zefixClient.FindCompany(companyName)
		if err != nil {
			log.Warnf("error while fetching company: %s", err)
			continue
		}

		if c != nil {
			companiesMap[c.LegalName] = *c
			log.Infof("Adding company: %v", c.LegalName)
		}
	}

	var companies []zefix.Company
	for _, c := range companiesMap {
		companies = append(companies, c)
	}

	return companies
}

func (p *Processor) Ping() error {
	return p.zefixClient.Ping()
}

func printErrors(errors []error) {
	if len(errors) != 0 {
		log.Warnf("found %d errors", len(errors))
		for _, err := range errors {
			log.Warnf("error: %s", err)
		}
	}
}
