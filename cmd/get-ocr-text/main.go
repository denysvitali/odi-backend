package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/sirupsen/logrus"

	"github.com/denysvitali/odi-backend/pkg/ocrclient"
	"github.com/denysvitali/odi-backend/pkg/ocrtext"
)

var args struct {
	InputFile          string  `arg:"positional,required"`
	MergeDistance      float64 `arg:"-d,--merge-distance" default:"150"`
	HorizontalDistance float64 `arg:"-D,--horizontal-distance" default:"10"`
}

var log = logrus.New()

func main() {
	arg.MustParse(&args)
	v, err := parseJson(args.InputFile)
	if err != nil {
		log.Fatalf("unable to parse JSON: %v", err)
	}
	text := ocrtext.GetText(v, args.MergeDistance, args.HorizontalDistance)
	fmt.Println(text)
}

func parseJson(file string) (*ocrclient.OcrResult, error) {
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
