package server

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
	"github.com/sirupsen/logrus"

	"github.com/denysvitali/odi-backend/pkg/models"
	"github.com/denysvitali/odi-backend/pkg/storage/model"
)

const (
	serverReadTimeout  = 30 * time.Second
	serverWriteTimeout = 30 * time.Second
)

type Server struct {
	e                    *gin.Engine
	osUrl                *url.URL
	osUsername           string
	osPassword           string
	osIndex              string
	osInsecureSkipVerify bool
	osClient             *opensearch.Client
	storage              model.Retriever
}

var log = logrus.StandardLogger().WithField("package", "server")

func New(osAddr string, osUsername string, osPassword string, osInsecureSkipVerify bool, osIndex string, ret model.Retriever) (*Server, error) {
	u, err := url.Parse(osAddr)
	if err != nil {
		return nil, err
	}

	s := Server{
		e:                    gin.New(),
		osUrl:                u,
		osUsername:           osUsername,
		osPassword:           osPassword,
		osInsecureSkipVerify: osInsecureSkipVerify,
		osIndex:              osIndex,
		storage:              ret,
	}

	var transport http.RoundTripper
	if s.osInsecureSkipVerify {
		transport = &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	} else {
		transport = http.DefaultTransport
	}

	c, err := opensearch.NewClient(
		opensearch.Config{
			Addresses: []string{s.osUrl.String()},
			Username:  s.osUsername,
			Password:  s.osPassword,
			Transport: transport,
		},
	)
	if err != nil {
		return nil, err
	}
	s.osClient = c

	err = s.verifyOpensearch(context.Background(), osIndex)
	if err != nil {
		return nil, err
	}

	s.initRoutes()
	return &s, nil
}

func (s *Server) verifyOpensearch(ctx context.Context, osIndex string) error {
	err := s.pingOs(ctx)
	if err != nil {
		return err
	}

	err = s.verifyIndex(ctx, osIndex)
	if err != nil {
		return err
	}
	return nil
}

func (s *Server) Run(addr string) error {
	srv := &http.Server{
		Addr:         addr,
		Handler:      s.e,
		ReadTimeout:  serverReadTimeout,
		WriteTimeout: serverWriteTimeout,
	}
	return srv.ListenAndServe()
}

func (s *Server) initRoutes() {
	s.e.Use(gin.Logger())
	s.e.Use(cors.Default())

	g := s.e.Group("/api/v1")
	g.POST("/search", s.handleSearch)
	g.GET("/documents/:id", s.handleGetDocument)
	g.GET("/documents", s.handleGetDocuments)
	g.GET("/files/:scanID/:sequenceId", s.handleGetFile)
	g.GET("/thumbnails/:id", s.handleGetThumbnail)
}

type SearchRequest struct {
	SearchTerm string `json:"searchTerm"`
	Size       int    `json:"size,omitempty"`
	ScrollId   string `json:"scrollId,omitempty"`
}

func (s *Server) handleSearch(c *gin.Context) {
	var searchRequest SearchRequest
	err := c.BindJSON(&searchRequest)
	if err != nil {
		c.JSON(http.StatusBadRequest, badRequest)
		return
	}

	var res *opensearchapi.Response

	if searchRequest.ScrollId != "" {
		// Continue pagination using scroll
		scrollBody := map[string]any{
			"scroll":    "10m",
			"scroll_id": searchRequest.ScrollId,
		}
		jsonBody, _ := json.Marshal(scrollBody)
		req := opensearchapi.ScrollRequest{
			Body: bytes.NewReader(jsonBody),
		}
		res, err = req.Do(c.Request.Context(), s.osClient)
	} else {
		// Initial search with scroll enabled
		size := searchRequest.Size
		if size <= 0 {
			size = 50
		}

		searchContent := map[string]any{
			"size": size,
			"query": map[string]any{
				"query_string": map[string]any{
					"query": searchRequest.SearchTerm,
				},
			},
			"highlight": map[string]any{
				"fields": map[string]any{
					"text": map[string]any{},
				},
			},
		}

		jsonBody, marshalErr := json.Marshal(searchContent)
		if marshalErr != nil {
			log.Errorf("unable to marshal JSON: %v", marshalErr)
			c.JSON(http.StatusInternalServerError, internalServerError)
			return
		}

		req := opensearchapi.SearchRequest{
			Index:  []string{s.osIndex},
			Body:   bytes.NewReader(jsonBody),
			Scroll: 10 * time.Minute,
		}
		res, err = req.Do(c.Request.Context(), s.osClient)
	}

	if err != nil {
		log.Errorf("unable to perform search: %v", err)
		c.JSON(http.StatusInternalServerError, internalServerError)
		return
	}

	if res.IsError() {
		log.Errorf("unable to perform search: %s", res.Status())
		c.JSON(http.StatusInternalServerError, internalServerError)
		return
	}

	c.Status(http.StatusOK)
	c.Header("Content-Type", "application/json")
	_, err = io.Copy(c.Writer, res.Body)
	if err != nil {
		log.Errorf("unable to copy: %v", err)
		return
	}
}

func (s *Server) returnDocument(c *gin.Context, scanID string, sequenceIdStr string) {
	sequenceId, err := strconv.ParseInt(sequenceIdStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, badRequest)
		return
	}

	page, err := s.storage.Retrieve(c.Request.Context(), scanID, int(sequenceId))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "not found",
			})
			return
		}
		log.Errorf("unable to retrieve page: %v", err)
		c.JSON(http.StatusInternalServerError, internalServerError)
		return
	}

	c.Header("Content-Type", "image/jpeg")
	c.Status(http.StatusOK)
	_, err = io.Copy(c.Writer, page.Reader)
	if err != nil {
		log.Errorf("unable to copy: %v", err)
		return
	}
}

func (s *Server) handleGetFile(c *gin.Context) {
	scanID := c.Param("scanID")
	sequenceIdStr := c.Param("sequenceId")

	if scanID == "" || sequenceIdStr == "" {
		c.JSON(http.StatusBadRequest, badRequest)
		return
	}

	s.returnDocument(c, scanID, sequenceIdStr)
}

func (s *Server) handleGetThumbnail(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, badRequest)
		return
	}

	// Parse id in format: scanID_sequenceID
	matches := docIdRegexp.FindStringSubmatch(id)
	if len(matches) != 3 {
		c.JSON(http.StatusBadRequest, badRequest)
		return
	}

	scanID := matches[1]
	sequenceIdStr := matches[2]
	s.returnDocument(c, scanID, sequenceIdStr)
}

var badRequest = gin.H{
	"error": "bad request",
}

var internalServerError = gin.H{
	"error": "internal server error",
}

type Document[T any] struct {
	Index       string `json:"_index"`
	Id          string `json:"_id"`
	Version     int    `json:"_version"`
	SeqNo       int    `json:"_seq_no"`
	PrimaryTerm int    `json:"_primary_term"`
	Found       bool   `json:"found"`
	Source      T      `json:"_source"`
}

var docIdRegexp = regexp.MustCompile("^([0-9a-f-]+)_([0-9]+)$")

func (s *Server) handleGetDocument(c *gin.Context) {
	docId := c.Param("id")
	if docId == "" {
		c.JSON(http.StatusBadRequest, badRequest)
		return
	}

	if !docIdRegexp.MatchString(docId) {
		c.JSON(http.StatusBadRequest, badRequest)
		return
	}

	req := opensearchapi.GetRequest{Index: s.osIndex, DocumentID: docId}
	res, err := req.Do(c.Request.Context(), s.osClient)
	if err != nil {
		c.JSON(http.StatusInternalServerError, internalServerError)
		return
	}

	if res.IsError() {
		log.Warnf("unable to get document %s: %s", docId, res.Status())
		c.JSON(http.StatusInternalServerError, internalServerError)
		return
	}

	var doc Document[models.Document]
	err = json.NewDecoder(res.Body).Decode(&doc)
	if err != nil {
		log.Errorf("unable to decode document: %v", err)
		c.JSON(http.StatusInternalServerError, internalServerError)
		return
	}

	if !doc.Found {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "not found",
		})
		return
	}

	c.JSON(http.StatusOK, doc.Source)
}

func (s *Server) handleGetDocuments(c *gin.Context) {
	scrollId := c.Query("scroll_id")
	var res *opensearchapi.Response
	var err error
	if scrollId != "" {
		// Use Body to pass scroll_id instead of URL query parameter to avoid 405 errors
		// when scroll_id is long. See https://github.com/opensearch-project/opensearch-go/issues/422
		scrollBody := map[string]any{
			"scroll":    "10m",
			"scroll_id": scrollId,
		}
		jsonBody, _ := json.Marshal(scrollBody)
		req := opensearchapi.ScrollRequest{
			Body: bytes.NewReader(jsonBody),
		}
		res, err = req.Do(c.Request.Context(), s.osClient)
	} else {
		// Use Body for sort and scroll to minimize URL parameters
		searchBody := map[string]any{
			"sort": []map[string]any{
				{"indexedAt": "desc"},
			},
		}
		jsonBody, _ := json.Marshal(searchBody)
		req := opensearchapi.SearchRequest{
			Index:  []string{s.osIndex},
			Body:   bytes.NewReader(jsonBody),
			Scroll: 10 * time.Minute,
		}
		res, err = req.Do(c.Request.Context(), s.osClient)
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, internalServerError)
		return
	}

	if res.IsError() {
		log.Warnf("unable to get documents: %s", res.Status())
		c.JSON(http.StatusInternalServerError, internalServerError)
		return
	}

	var docs struct {
		Hits struct {
			Hits []Document[models.Document] `json:"hits"`
		} `json:"hits"`
		ScrollId string `json:"_scroll_id"`
	}
	err = json.NewDecoder(res.Body).Decode(&docs)
	if err != nil {
		log.Errorf("unable to decode documents: %v", err)
		c.JSON(http.StatusInternalServerError, internalServerError)
		return
	}

	c.JSON(http.StatusOK, docs)
}

func (s *Server) pingOs(ctx context.Context) error {
	req := opensearchapi.PingRequest{}
	res, err := req.Do(ctx, s.osClient)
	if err != nil {
		return err
	}
	if res.IsError() {
		return fmt.Errorf("unable to ping OS: %s", res.Status())
	}
	return nil
}

func (s *Server) verifyIndex(ctx context.Context, index string) error {
	req := opensearchapi.CatIndicesRequest{Index: []string{index}}
	res, err := req.Do(ctx, s.osClient)
	if err != nil {
		return err
	}

	if res.IsError() {
		return fmt.Errorf("unable to verify index %s: %s", index, res.Status())
	}
	return nil
}
