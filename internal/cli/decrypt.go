package cli

import (
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/denysvitali/odi-backend/internal/ui"
	odicrypt "github.com/denysvitali/odi-backend/pkg/crypt"
)

const (
	FlagPassphrase = "passphrase"
)

var decryptCmd = &cobra.Command{
	Use:   "decrypt",
	Short: "Decrypt encrypted file content",
	Long: `Decrypt encrypted file content from stdin.

This command reads encrypted data from stdin and writes the
decrypted content to stdout. The passphrase can be provided
via the PASSPHRASE environment variable or the --passphrase flag.

Example:
  cat encrypted.bin | odi decrypt > decrypted.txt
  odi decrypt --passphrase="secret" < encrypted.bin > decrypted.txt`,
	RunE: runDecrypt,
}

func init() {
	decryptCmd.Flags().String(FlagPassphrase, "", "Decryption passphrase (env: PASSPHRASE)")

	bindEnv(FlagPassphrase, "PASSPHRASE")
}

func runDecrypt(cmd *cobra.Command, args []string) error {
	passphrase := GetString(cmd, FlagPassphrase)
	if passphrase == "" {
		ui.PrintError("Passphrase cannot be empty. Set via --passphrase or PASSPHRASE env var")
		return nil
	}

	c, err := odicrypt.New(passphrase)
	if err != nil {
		ui.PrintErrorf("Failed to create decryptor: %v", err)
		return err
	}

	reader, err := c.Decrypt(os.Stdin)
	if err != nil {
		ui.PrintErrorf("Decryption failed: %v", err)
		return err
	}

	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		ui.PrintErrorf("Failed to write output: %v", err)
		return err
	}

	return nil
}
