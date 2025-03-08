package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/sirupsen/logrus"

	"github.com/denysvitali/odi-backend/pkg/ocrclient"
	"github.com/denysvitali/odi-backend/pkg/ocrclient/caroundtripper"
)

var args struct {
	InputPath string `arg:"positional,required"`

	Debug        *bool  `arg:"-D,--debug"`
	OcrApi       string `arg:"-a,--ocr-api,env:OCR_API_ADDR,required" help:"Address of the OCR API"`
	OcrApiCaPath string `arg:"-c,--ocr-api-ca-path,env:OCR_API_CA_PATH"`
	OutputMode   string `arg:"-o,--output-mode" default:"text"`
}

var log = logrus.New()

func main() {
	arg.MustParse(&args)
	if args.Debug != nil && *args.Debug {
		log.SetLevel(logrus.DebugLevel)
	}

	c, err := ocrclient.New(args.OcrApi)
	if err != nil {
		log.Fatalf("unable to create client: %v", err)
	}

	if args.OcrApiCaPath != "" {
		rt, err := caroundtripper.New(args.OcrApiCaPath)
		if err != nil {
			log.Fatalf("unable to create CA Roundtripper: %v", err)
		}
		c.SetHttpTransport(rt)
	}

	f, err := os.Open(args.InputPath)
	if err != nil {
		log.Fatalf("unable to open file: %v", err)
	}
	defer f.Close()
	res, err := c.Process(f)
	if err != nil {
		log.Errorf("unable to process file: %v", err)
		return
	}

	switch args.OutputMode {
	case "json":
		e := json.NewEncoder(os.Stdout)
		err = e.Encode(res)
		if err != nil {
			log.Fatalf("unable to encode JSON: %v", err)
		}
	default:
		fmt.Print(res.Text())
	}
}
