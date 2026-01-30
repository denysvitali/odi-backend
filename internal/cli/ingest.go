package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/denysvitali/odi-backend/internal/ui"
	"github.com/denysvitali/odi-backend/pkg/ingestor"
)

const (
	FlagScannerName = "scanner-name"
	FlagSource      = "source"
)

var ingestCmd = &cobra.Command{
	Use:   "ingest",
	Short: "Ingest documents from a network scanner",
	Long: `Ingest documents from a network scanner via AirScan protocol.

This command will:
  1. Connect to the specified scanner
  2. Scan documents
  3. Store them in the configured storage backend
  4. Index them via OCR processing`,
	RunE: runIngest,
}

func init() {
	ingestCmd.Flags().String(FlagScannerName, "", "Name of the scanner to use (env: SCANNER_NAME)")
	ingestCmd.Flags().String(FlagSource, "Feeder", "Document source: Feeder or Platen (env: SOURCE)")

	bindEnv(FlagScannerName, "SCANNER_NAME")
	bindEnv(FlagSource, "SOURCE")

	AddOpenSearchFlags(ingestCmd)
	AddStorageFlags(ingestCmd)
	AddOCRFlags(ingestCmd)
	AddZefixFlags(ingestCmd)

}

func runIngest(cmd *cobra.Command, args []string) error {
	if GetString(cmd, FlagScannerName) == "" {
		return fmt.Errorf("required flag or env var not set: %s (env: SCANNER_NAME)", FlagScannerName)
	}
	if GetString(cmd, FlagOsAddr) == "" {
		return fmt.Errorf("required flag or env var not set: %s (env: OPENSEARCH_ADDR)", FlagOsAddr)
	}
	if GetString(cmd, FlagOcrAPIAddr) == "" {
		return fmt.Errorf("required flag or env var not set: %s (env: OCR_API_ADDR)", FlagOcrAPIAddr)
	}
	if GetString(cmd, FlagZefixDsn) == "" {
		return fmt.Errorf("required flag or env var not set: %s (env: ZEFIX_DSN)", FlagZefixDsn)
	}
	if GetString(cmd, FlagStorageType) == "" {
		return fmt.Errorf("required flag or env var not set: %s (env: STORAGE_TYPE)", FlagStorageType)
	}

	storage, err := GetStorage(cmd)
	if err != nil {
		ui.PrintErrorf("Failed to initialize storage: %v", err)
		return err
	}
	if storage == nil {
		ui.PrintError("Invalid storage type. Must be 'b2' or 'fs'")
		return nil
	}

	scannerName := GetString(cmd, FlagScannerName)
	source := GetString(cmd, FlagSource)

	ui.PrintInfof("Connecting to scanner: %s", scannerName)

	i, err := ingestor.New(ingestor.Config{
		OcrAPIAddr:         GetString(cmd, FlagOcrAPIAddr),
		OpenSearchAddr:     GetString(cmd, FlagOsAddr),
		OpenSearchPassword: GetString(cmd, FlagOsPassword),
		OpenSearchSkipTLS:  GetBool(cmd, FlagOsSkipTLS),
		OpenSearchUsername: GetString(cmd, FlagOsUsername),
		Storage:            storage,
		ZefixDsn:           GetString(cmd, FlagZefixDsn),
	})
	if err != nil {
		ui.PrintErrorf("Failed to create ingestor: %v", err)
		return err
	}

	ctx := context.Background()
	ui.PrintInfo("Starting document ingestion...")
	err = i.Ingest(ctx, scannerName, source)
	if err != nil {
		ui.PrintErrorf("Ingestion failed: %v", err)
		return err
	}

	ui.PrintSuccess("Ingestion completed successfully")
	return nil
}
