package ocrclient

import (
	"bytes"
	"fmt"
	"sort"
)

type TextBlock struct {
	Text        string      `json:"text"`
	Lines       []line      `json:"lines"`
	BoundingBox BoundingBox `json:"boundingBox"`
	Lang        string      `json:"lang"`
}

type line struct {
	Text               string  `json:"text"`
	Angle              float64 `json:"angle"`
	Confidence         float64 `json:"confidence"`
	RecognizedLanguage string  `json:"recognizedLanguage"`
}

type BoundingBox struct {
	Top    int `json:"top"`
	Bottom int `json:"bottom"`
	Left   int `json:"left"`
	Right  int `json:"right"`
}

type barcode struct {
	BoundingBox  BoundingBox `json:"boundingBox"`
	DisplayValue string      `json:"displayValue"`
	RawValue     string      `json:"rawValue"`
}

type OcrResult struct {
	TextBlocks []TextBlock `json:"textBlocks"`
	Barcodes   []barcode   `json:"barcodes"`
}

type SortText []TextBlock

func (s SortText) Len() int {
	return len(s)
}

func (s SortText) Less(i, j int) bool {
	bbI := s[i].BoundingBox
	bbJ := s[j].BoundingBox
	if bbI.Top < bbJ.Top {
		return true
	} else if bbI.Top == bbJ.Top {
		if bbI.Left <= bbJ.Left {
			return true
		}
		return false
	} else {
		return false
	}
}

func (s SortText) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

var _ sort.Interface = SortText{}

func (o *OcrResult) Text() string {
	blocks := o.TextBlocks
	sort.Sort(SortText(blocks))
	groups := GroupTextBlocks(blocks, 5, 200)

	buffer := bytes.NewBuffer(nil)
	for _, b := range groups {
		for _, v := range b {
			fmt.Fprintf(buffer, "%s\n", v.Text)
		}

		fmt.Fprintf(buffer, "\n")
	}
	return buffer.String()
}
