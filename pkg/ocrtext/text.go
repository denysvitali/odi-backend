package ocrtext

import (
	"fmt"
	"math"
	"sort"

	"github.com/denysvitali/odi-backend/pkg/ocrclient"
)

// GetText returns the text from the OCR result
// sorted in a way that matches the document text
// order
func GetText(v *ocrclient.OcrResult, mergeDistance float64, horizontalDistance float64) string {
	columns := map[int][]ocrclient.TextBlock{}
	for _, b := range v.TextBlocks {
		left := b.BoundingBox.Left

		found := false
		for k, v := range columns {
			if math.Abs(float64(k-left)) < mergeDistance {
				columns[k] = append(v, b)
				found = true
				break
			}
		}

		if !found {
			columns[left] = []ocrclient.TextBlock{b}
		}
	}

	for _, c := range columns {
		// Sort by top
		sort.Slice(c, func(i, j int) bool {
			// If the top diff is less than 5, sort by left
			if math.Abs(float64(c[i].BoundingBox.Top-c[j].BoundingBox.Top)) <= horizontalDistance {
				return c[i].BoundingBox.Left < c[j].BoundingBox.Left
			}

			return c[i].BoundingBox.Top < c[j].BoundingBox.Top
		})
	}

	columnsValue := make([][]ocrclient.TextBlock, 0, len(columns))
	for _, v := range columns {
		columnsValue = append(columnsValue, v)
	}

	sort.Slice(columnsValue, func(i, j int) bool {
		if columns[i] == nil || columns[j] == nil {
			return false
		}
		if math.Abs(float64(columns[i][0].BoundingBox.Top-columns[j][0].BoundingBox.Top)) <= horizontalDistance {
			return columns[i][0].BoundingBox.Left < columns[j][0].BoundingBox.Left
		}
		return columns[i][0].BoundingBox.Top < columns[j][0].BoundingBox.Top
	})

	output := ""

	// Print the text
	for _, c := range columnsValue {
		var prevBlock *ocrclient.TextBlock = nil
		for _, b := range c {
			currentBlock := b
			if prevBlock != nil {
				if math.Abs(float64(prevBlock.BoundingBox.Top-b.BoundingBox.Top)) >= horizontalDistance {
					output += fmt.Sprintf("\n\n")
				} else {
					output += fmt.Sprintf(" ")
				}
			}
			output += fmt.Sprintf("%s", b.Text)
			prevBlock = &currentBlock
		}
		output += fmt.Sprintf("\n")
	}

	return output
}
