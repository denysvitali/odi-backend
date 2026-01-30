package cli

import (
	"context"
	"fmt"
	"os"
	"path"
	"sync"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/denysvitali/odi-backend/internal/ui"
	"github.com/denysvitali/odi-backend/pkg/indexer"
	"github.com/denysvitali/odi-backend/pkg/models"
)

const (
	FlagInputDir          = "input-dir"
	FlagWorkers           = "workers"
	FlagDebug             = "debug"
	DefaultIndexWorkers   = 4
)

var indexCmd = &cobra.Command{
	Use:   "index [input-dir]",
	Short: "Index documents from a directory",
	Long: `Index documents from a local directory.

This command will:
  1. Read all image files from the specified directory
  2. Process them through OCR
  3. Extract text and metadata (companies, dates, barcodes)
  4. Store the indexed documents in OpenSearch`,
	Args: cobra.ExactArgs(1),
	RunE: runIndex,
}

func init() {
	indexCmd.Flags().IntP(FlagWorkers, "w", DefaultIndexWorkers, "Number of worker goroutines")
	indexCmd.Flags().BoolP(FlagDebug, "D", false, "Enable debug logging")

	AddOpenSearchFlags(indexCmd)
	AddOCRFlags(indexCmd)
	AddZefixFlags(indexCmd)

}

func runIndex(cmd *cobra.Command, args []string) error {
	if GetString(cmd, FlagOsAddr) == "" {
		return fmt.Errorf("required flag or env var not set: %s (env: OPENSEARCH_ADDR)", FlagOsAddr)
	}
	if GetString(cmd, FlagOsUsername) == "" {
		return fmt.Errorf("required flag or env var not set: %s (env: OPENSEARCH_USERNAME)", FlagOsUsername)
	}
	if GetString(cmd, FlagOsPassword) == "" {
		return fmt.Errorf("required flag or env var not set: %s (env: OPENSEARCH_PASSWORD)", FlagOsPassword)
	}
	if GetString(cmd, FlagOcrAPIAddr) == "" {
		return fmt.Errorf("required flag or env var not set: %s (env: OCR_API_ADDR)", FlagOcrAPIAddr)
	}
	if GetString(cmd, FlagZefixDsn) == "" {
		return fmt.Errorf("required flag or env var not set: %s (env: ZEFIX_DSN)", FlagZefixDsn)
	}

	log := logrus.StandardLogger()
	inputDir := args[0]

	debug := GetBool(cmd, FlagDebug)
	if debug {
		log.SetLevel(logrus.DebugLevel)
	}

	workers := GetInt(cmd, FlagWorkers)
	if workers <= 0 {
		workers = DefaultIndexWorkers
		ui.PrintWarningf("Workers cannot be <= 0, using default: %d", workers)
	}

	opts := []indexer.Option{
		indexer.WithOpenSearchUsername(GetString(cmd, FlagOsUsername)),
		indexer.WithOpenSearchPassword(GetString(cmd, FlagOsPassword)),
		indexer.WithOcrApiCAPath(GetString(cmd, FlagOcrCaPath)),
	}
	if GetBool(cmd, FlagOsSkipTLS) {
		opts = append(opts, indexer.WithOpenSearchSkipTLS())
	}

	idx, err := indexer.New(
		GetString(cmd, FlagOsAddr),
		GetString(cmd, FlagOcrAPIAddr),
		GetString(cmd, FlagZefixDsn),
		opts...,
	)
	if err != nil {
		ui.PrintErrorf("Failed to create indexer: %v", err)
		return err
	}

	ctx := context.Background()
	res, err := idx.PingOpensearch(ctx)
	if err != nil {
		ui.PrintErrorf("Failed to ping OpenSearch: %v", err)
		return err
	}

	if res.IsError() {
		ui.PrintErrorf("OpenSearch ping failed: %s", res.Status())
		return nil
	}
	ui.PrintSuccess("Connected to OpenSearch")

	ch := make(chan models.ScannedPage, workers)
	wg := sync.WaitGroup{}

	for i := 0; i < workers; i++ {
		wg.Add(1)
		w := indexer.NewWorker(i, ch)
		w.SetIndexer(idx)
		go w.Start(ctx, &wg)
	}

	scanID := uuid.NewString()
	seq := 0

	files, err := os.ReadDir(inputDir)
	if err != nil {
		ui.PrintErrorf("Failed to read directory: %v", err)
		return err
	}

	ui.PrintInfof("Processing %d files from %s", len(files), inputDir)

	for _, file := range files {
		if !file.IsDir() {
			seq++
			f, err := os.Open(path.Join(inputDir, file.Name()))
			if err != nil {
				log.Errorf("Unable to open file: %v", err)
				continue
			}
			ch <- models.ScannedPage{
				Reader:     f,
				ScanID:     scanID,
				SequenceID: seq,
			}
		}
	}
	close(ch)
	wg.Wait()

	ui.PrintSuccessf("Indexing complete. Scan ID: %s", scanID)
	return nil
}
