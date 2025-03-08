package ocrclient_test

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"os"
	"sort"
	"testing"

	"gocv.io/x/gocv"

	"github.com/denysvitali/odi-backend/pkg/ocrclient"
)

var redColor = color.RGBA{R: 255, G: 0, B: 0, A: 255}
var greenColor = color.RGBA{R: 0, G: 255, B: 0, A: 255}

func TestSort(t *testing.T) {
	if os.Getenv("E2E_TEST") != "true" {
		t.Skip("skipping test; E2E_TEST is not set")
	}
	f, err := os.Open("../../resources/testdata/ocr/private/1.json")
	if err != nil {
		t.Fatalf("unable to open JSON: %v", err)
	}

	var ocrResult ocrclient.OcrResult
	dec := json.NewDecoder(f)
	err = dec.Decode(&ocrResult)
	if err != nil {
		t.Fatalf("unable to decode JSON: %v", err)
	}

	img := gocv.IMRead("../../resources/testdata/ocr/private/1.jpg", gocv.IMReadColor)

	sort.Sort(ocrclient.SortText(ocrResult.TextBlocks))
	groups := ocrclient.GroupTextBlocks(ocrResult.TextBlocks, 5, 200)

	for idx, b := range groups {
		bb := ocrclient.TextBlockGroup(b).BoundingBox()
		drawBB(&img, bb, idx)

		for _, v := range b {
			fmt.Printf("%s\n", v.Text)
		}

		fmt.Printf("\n")
	}

	window := gocv.NewWindow("Result")
	window.IMShow(img)
	window.WaitKey(-1)

}

func drawBB(img *gocv.Mat, bb ocrclient.BoundingBox, idx int) {
	gocv.Rectangle(img,
		image.Rect(bb.Left, bb.Top, bb.Right, bb.Bottom),
		redColor,
		1.0,
	)
	gocv.PutText(img,
		fmt.Sprintf("%d", idx),
		image.Point{X: bb.Left, Y: bb.Top},
		gocv.FontHersheySimplex,
		1.0,
		greenColor,
		3,
	)
}
