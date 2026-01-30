package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	cliutils "github.com/denysvitali/odi-backend/pkg/cli"
	"github.com/denysvitali/odi-backend/pkg/storage"
	"github.com/denysvitali/odi-backend/pkg/storage/b2"
	"github.com/denysvitali/odi-backend/pkg/storage/model"
)

// Flag names as constants for consistency
const (
	// Global flags
	FlagLogLevel = "log-level"
	FlagConfig   = "config"

	// OpenSearch flags
	FlagOsAddr     = "opensearch-addr"
	FlagOsUsername = "opensearch-username"
	FlagOsPassword = "opensearch-password"
	FlagOsSkipTLS  = "opensearch-skip-tls"
	FlagOsIndex    = "opensearch-index"

	// Storage flags
	FlagStorageType  = "storage-type"
	FlagB2AccountID  = "b2-account-id"
	FlagB2AccountKey = "b2-account-key"
	FlagB2BucketName = "b2-bucket-name"
	FlagB2Passphrase = "b2-passphrase"
	FlagFsPath       = "fs-path"

	// OCR flags
	FlagOcrAPIAddr = "ocr-api-addr"
	FlagOcrCaPath  = "ocr-api-ca-path"

	// Zefix flags
	FlagZefixDsn = "zefix-dsn"
)

// AddOpenSearchFlags adds OpenSearch connection flags to a command
func AddOpenSearchFlags(cmd *cobra.Command) {
	cmd.Flags().String(FlagOsAddr, "", "OpenSearch address (env: OPENSEARCH_ADDR)")
	cmd.Flags().String(FlagOsUsername, "", "OpenSearch username (env: OPENSEARCH_USERNAME)")
	cmd.Flags().String(FlagOsPassword, "", "OpenSearch password (env: OPENSEARCH_PASSWORD)")
	cmd.Flags().Bool(FlagOsSkipTLS, false, "Skip TLS verification for OpenSearch (env: OPENSEARCH_SKIP_TLS)")
	cmd.Flags().String(FlagOsIndex, "documents", "OpenSearch index name (env: OPENSEARCH_INDEX)")

	bindEnv(FlagOsAddr, "OPENSEARCH_ADDR")
	bindEnv(FlagOsUsername, "OPENSEARCH_USERNAME")
	bindEnv(FlagOsPassword, "OPENSEARCH_PASSWORD")
	bindEnv(FlagOsSkipTLS, "OPENSEARCH_SKIP_TLS")
	bindEnv(FlagOsIndex, "OPENSEARCH_INDEX")
}

// AddStorageFlags adds storage backend flags to a command
func AddStorageFlags(cmd *cobra.Command) {
	cmd.Flags().String(FlagStorageType, "", "Storage type: b2 or fs (env: STORAGE_TYPE)")
	cmd.Flags().String(FlagB2AccountID, "", "B2 account ID (env: B2_ACCOUNT)")
	cmd.Flags().String(FlagB2AccountKey, "", "B2 account key (env: B2_KEY)")
	cmd.Flags().String(FlagB2BucketName, "", "B2 bucket name (env: B2_BUCKET_NAME)")
	cmd.Flags().String(FlagB2Passphrase, "", "B2 encryption passphrase (env: B2_PASSPHRASE)")
	cmd.Flags().String(FlagFsPath, "", "Filesystem storage path (env: FS_PATH)")

	bindEnv(FlagStorageType, "STORAGE_TYPE")
	bindEnv(FlagB2AccountID, "B2_ACCOUNT")
	bindEnv(FlagB2AccountKey, "B2_KEY")
	bindEnv(FlagB2BucketName, "B2_BUCKET_NAME")
	bindEnv(FlagB2Passphrase, "B2_PASSPHRASE")
	bindEnv(FlagFsPath, "FS_PATH")
}

// AddOCRFlags adds OCR API flags to a command
func AddOCRFlags(cmd *cobra.Command) {
	cmd.Flags().String(FlagOcrAPIAddr, "", "OCR API address (env: OCR_API_ADDR)")
	cmd.Flags().String(FlagOcrCaPath, "", "Path to CA certificate for OCR API (env: OCR_API_CA_PATH)")

	bindEnv(FlagOcrAPIAddr, "OCR_API_ADDR")
	bindEnv(FlagOcrCaPath, "OCR_API_CA_PATH")
}

// AddZefixFlags adds Zefix database flags to a command
func AddZefixFlags(cmd *cobra.Command) {
	cmd.Flags().String(FlagZefixDsn, "", "Zefix database DSN (env: ZEFIX_DSN)")

	bindEnv(FlagZefixDsn, "ZEFIX_DSN")
}

// bindEnv binds a flag to environment variables
func bindEnv(flagName string, envVars ...string) {
	allVars := append([]string{flagName}, envVars...)
	_ = viper.BindEnv(allVars...)
}

// GetString returns a string value from viper, checking flag then env
func GetString(cmd *cobra.Command, flagName string) string {
	if cmd.Flags().Changed(flagName) {
		val, err := cmd.Flags().GetString(flagName)
		if err != nil {
			return ""
		}
		return val
	}
	// Check viper (env var), but if it's empty, use the flag's default value
	val := viper.GetString(flagName)
	if val == "" {
		val, _ = cmd.Flags().GetString(flagName)
	}
	return val
}

// GetBool returns a bool value from viper, checking flag then env
func GetBool(cmd *cobra.Command, flagName string) bool {
	if cmd.Flags().Changed(flagName) {
		val, err := cmd.Flags().GetBool(flagName)
		if err != nil {
			return false
		}
		return val
	}
	return viper.GetBool(flagName)
}

// GetInt returns an int value from viper, checking flag then env
func GetInt(cmd *cobra.Command, flagName string) int {
	if cmd.Flags().Changed(flagName) {
		val, err := cmd.Flags().GetInt(flagName)
		if err != nil {
			return 0
		}
		return val
	}
	return viper.GetInt(flagName)
}

// GetFloat64 returns a float64 value from viper, checking flag then env
func GetFloat64(cmd *cobra.Command, flagName string) float64 {
	if cmd.Flags().Changed(flagName) {
		val, err := cmd.Flags().GetFloat64(flagName)
		if err != nil {
			return 0
		}
		return val
	}
	return viper.GetFloat64(flagName)
}

// GetStorage returns a configured storage backend based on flags
func GetStorage(cmd *cobra.Command) (model.RWStorage, error) {
	storageType := GetString(cmd, FlagStorageType)
	switch strings.ToLower(storageType) {
	case "b2":
		config := b2.Config{
			Account:    GetString(cmd, FlagB2AccountID),
			BucketName: GetString(cmd, FlagB2BucketName),
			Key:        GetString(cmd, FlagB2AccountKey),
			Passphrase: GetString(cmd, FlagB2Passphrase),
		}
		if err := cliutils.FillKeychainValues(&config); err != nil {
			return nil, fmt.Errorf("failed to resolve keychain values: %w", err)
		}
		return storage.SetupB2Storage(config)
	case "fs":
		return storage.SetupFsStorage(GetString(cmd, FlagFsPath))
	default:
		return nil, nil
	}
}
