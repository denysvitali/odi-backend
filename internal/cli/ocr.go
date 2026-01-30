package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/denysvitali/odi-backend/internal/ui"
	"github.com/denysvitali/odi-backend/pkg/ocrclient"
	"github.com/denysvitali/odi-backend/pkg/ocrclient/caroundtripper"
)

const (
	FlagInput      = "input"
	FlagOutputMode = "output-mode"
)

var ocrCmd = &cobra.Command{
	Use:   "ocr [input-file]",
	Short: "Process an image through OCR",
	Long: `Process an image file through the OCR API.

Output modes:
  - text: Plain text extracted from the image (default)
  - json: Full OCR result including bounding boxes and barcodes`,
	Args: cobra.ExactArgs(1),
	RunE: runOcr,
}

func init() {
	ocrCmd.Flags().StringP(FlagOutputMode, "o", "text", "Output mode: text or json")

	AddOCRFlags(ocrCmd)

}

func runOcr(cmd *cobra.Command, args []string) error {
	if GetString(cmd, FlagOcrAPIAddr) == "" {
		return fmt.Errorf("required flag or env var not set: %s (env: OCR_API_ADDR)", FlagOcrAPIAddr)
	}

	log := logrus.StandardLogger()
	inputPath := args[0]

	debug := GetBool(cmd, FlagDebug)
	if debug {
		log.SetLevel(logrus.DebugLevel)
	}

	c, err := ocrclient.New(GetString(cmd, FlagOcrAPIAddr))
	if err != nil {
		ui.PrintErrorf("Failed to create OCR client: %v", err)
		return err
	}

	caPath := GetString(cmd, FlagOcrCaPath)
	if caPath != "" {
		rt, err := caroundtripper.New(caPath)
		if err != nil {
			ui.PrintErrorf("Failed to create CA RoundTripper: %v", err)
			return err
		}
		c.SetHTTPTransport(rt)
	}

	f, err := os.Open(inputPath)
	if err != nil {
		ui.PrintErrorf("Failed to open file: %v", err)
		return err
	}
	defer f.Close()

	res, err := c.Process(f)
	if err != nil {
		ui.PrintErrorf("OCR processing failed: %v", err)
		return err
	}

	outputMode := GetString(cmd, FlagOutputMode)
	switch outputMode {
	case "json":
		e := json.NewEncoder(os.Stdout)
		e.SetIndent("", "  ")
		err = e.Encode(res)
		if err != nil {
			ui.PrintErrorf("Failed to encode JSON: %v", err)
			return err
		}
	default:
		fmt.Print(res.Text())
	}

	return nil
}
