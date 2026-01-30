package cli

import (
	"fmt"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/denysvitali/odi-backend/internal/server"
	"github.com/denysvitali/odi-backend/internal/ui"
)

const (
	FlagListenAddr = "listen-addr"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the REST API server",
	Long: `Start the ODI REST API server for searching and retrieving documents.

The server provides the following endpoints:
  - POST /api/v1/search      - Search documents
  - GET  /api/v1/documents   - List documents
  - GET  /api/v1/documents/:id - Get document by ID
  - GET  /api/v1/files/:scanID/:sequenceId - Get document file`,
	RunE: runServe,
}

func init() {
	serveCmd.Flags().StringP(FlagListenAddr, "L", "127.0.0.1:8085", "Address to listen on")
	_ = viper.BindPFlag(FlagListenAddr, serveCmd.Flags().Lookup(FlagListenAddr))

	AddOpenSearchFlags(serveCmd)
	AddStorageFlags(serveCmd)

}

func runServe(cmd *cobra.Command, args []string) error {
	log := logrus.StandardLogger()

	if GetString(cmd, FlagOsAddr) == "" {
		return fmt.Errorf("required flag or env var not set: %s (env: OPENSEARCH_ADDR)", FlagOsAddr)
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

	listenAddr := GetString(cmd, FlagListenAddr)

	s, err := server.New(
		GetString(cmd, FlagOsAddr),
		GetString(cmd, FlagOsUsername),
		GetString(cmd, FlagOsPassword),
		GetBool(cmd, FlagOsSkipTLS),
		GetString(cmd, FlagOsIndex),
		storage,
	)
	if err != nil {
		ui.PrintErrorf("Failed to create server: %v", err)
		return err
	}

	ui.PrintSuccessf("Starting server on %s", listenAddr)
	err = s.Run(listenAddr)
	if err != nil {
		log.Fatalf("Server error: %v", err)
	}
	return nil
}
