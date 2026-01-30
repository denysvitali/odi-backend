package cli

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/denysvitali/odi-backend/internal/ui"
	cliutils "github.com/denysvitali/odi-backend/pkg/cli"
	"github.com/denysvitali/odi-backend/pkg/indexer"
	"github.com/denysvitali/odi-backend/pkg/storage/b2"
)

const (
	FlagScanID = "scan-id"
)

var reindexCmd = &cobra.Command{
	Use:   "reindex [scan-id]",
	Short: "Re-index documents from B2 storage",
	Long: `Re-index previously stored documents from B2 storage.

This is useful for:
  - Applying new indexing rules to existing documents
  - Re-indexing documents that failed during initial ingestion
  - Updating document metadata with improved OCR or extraction`,
	Args: cobra.ExactArgs(1),
	RunE: runReindex,
}

func init() {
	reindexCmd.Flags().String(FlagB2AccountID, "", "B2 account ID (env: B2_ACCOUNT)")
	reindexCmd.Flags().String(FlagB2AccountKey, "", "B2 account key (env: B2_KEY)")
	reindexCmd.Flags().String(FlagB2BucketName, "", "B2 bucket name (env: B2_BUCKET_NAME)")
	reindexCmd.Flags().String(FlagB2Passphrase, "", "B2 encryption passphrase (env: B2_PASSPHRASE)")

	bindEnv(FlagB2AccountID, "B2_ACCOUNT")
	bindEnv(FlagB2AccountKey, "B2_KEY")
	bindEnv(FlagB2BucketName, "B2_BUCKET_NAME")
	bindEnv(FlagB2Passphrase, "B2_PASSPHRASE")

	AddOpenSearchFlags(reindexCmd)
	AddOCRFlags(reindexCmd)
	AddZefixFlags(reindexCmd)

}

func runReindex(cmd *cobra.Command, args []string) error {
	if GetString(cmd, FlagOsAddr) == "" {
		return fmt.Errorf("required flag or env var not set: %s (env: OPENSEARCH_ADDR)", FlagOsAddr)
	}
	if GetString(cmd, FlagOcrAPIAddr) == "" {
		return fmt.Errorf("required flag or env var not set: %s (env: OCR_API_ADDR)", FlagOcrAPIAddr)
	}
	if GetString(cmd, FlagZefixDsn) == "" {
		return fmt.Errorf("required flag or env var not set: %s (env: ZEFIX_DSN)", FlagZefixDsn)
	}

	log := logrus.StandardLogger()
	scanID := args[0]

	ui.PrintInfof("Re-indexing scan: %s", scanID)

	config := b2.Config{
		Account:    GetString(cmd, FlagB2AccountID),
		Key:        GetString(cmd, FlagB2AccountKey),
		BucketName: GetString(cmd, FlagB2BucketName),
		Passphrase: GetString(cmd, FlagB2Passphrase),
	}
	if err := cliutils.FillKeychainValues(&config); err != nil {
		return fmt.Errorf("failed to resolve keychain values: %w", err)
	}
	b, err := b2.New(config)
	if err != nil {
		ui.PrintErrorf("Failed to create B2 storage: %v", err)
		return err
	}

	var opts []indexer.Option
	if username := GetString(cmd, FlagOsUsername); username != "" {
		opts = append(opts, indexer.WithOpenSearchUsername(username))
	}
	if password := GetString(cmd, FlagOsPassword); password != "" {
		opts = append(opts, indexer.WithOpenSearchPassword(password))
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

	scanFiles, err := b.ListFiles(scanID)
	if err != nil {
		ui.PrintErrorf("Failed to list files: %v", err)
		return err
	}

	ui.PrintInfof("Found %d files to re-index", len(scanFiles))

	ctx := context.Background()
	for _, f := range scanFiles {
		log.Infof("Indexing %s", f.ID())
		scannedPage, err := b.Retrieve(ctx, f.ScanID, f.SequenceID)
		if err != nil {
			log.Errorf("Failed to retrieve file %s: %v", f.ID(), err)
			continue
		}
		err = idx.Index(ctx, *scannedPage)
		if err != nil {
			log.Errorf("Failed to index file %s: %v", f.ID(), err)
			continue
		}
	}

	ui.PrintSuccess("Re-indexing complete")
	return nil
}
