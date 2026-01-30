package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/denysvitali/odi-backend/internal/ui"
	"github.com/denysvitali/odi-backend/pkg/ocrclient"
	"github.com/denysvitali/odi-backend/pkg/ocrtext"
)

const (
	FlagMergeDistance      = "merge-distance"
	FlagHorizontalDistance = "horizontal-distance"
)

var ocrTextCmd = &cobra.Command{
	Use:   "ocr-text [input-file]",
	Short: "Extract text from OCR JSON output",
	Long: `Convert raw OCR JSON output to formatted plain text.

This command processes the JSON output from the OCR API and applies
text merging and column-based ordering to produce readable text.

Parameters:
  - merge-distance: Vertical distance threshold for merging text blocks
  - horizontal-distance: Horizontal distance threshold for column detection`,
	Args: cobra.ExactArgs(1),
	RunE: runOcrText,
}

func init() {
	ocrTextCmd.Flags().Float64P(FlagMergeDistance, "d", 150, "Merge distance for text blocks")
	ocrTextCmd.Flags().Float64P(FlagHorizontalDistance, "D", 10, "Horizontal distance for column detection")
}

func runOcrText(cmd *cobra.Command, args []string) error {
	inputFile := args[0]

	mergeDistance := GetFloat64(cmd, FlagMergeDistance)
	horizontalDistance := GetFloat64(cmd, FlagHorizontalDistance)

	v, err := parseOcrJson(inputFile)
	if err != nil {
		ui.PrintErrorf("Failed to parse JSON: %v", err)
		return err
	}

	text := ocrtext.GetText(v, mergeDistance, horizontalDistance)
	fmt.Println(text)

	return nil
}

func parseOcrJson(file string) (*ocrclient.OcrResult, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	dec := json.NewDecoder(f)
	var v ocrclient.OcrResult
	err = dec.Decode(&v)
	return &v, err
}
