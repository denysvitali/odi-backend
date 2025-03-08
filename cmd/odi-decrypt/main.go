package main

import (
	"io"
	"os"

	"github.com/alexflint/go-arg"
	"github.com/sirupsen/logrus"

	odicrypt "github.com/denysvitali/odi-backend/pkg/crypt"
)

var args struct {
	Passphrase string `arg:"env:PASSPHRASE"`
}

var log = logrus.StandardLogger()

func main() {
	arg.MustParse(&args)

	if args.Passphrase == "" {
		log.Fatalf("passphrase cannot be empty")
	}

	c, err := odicrypt.New(args.Passphrase)
	if err != nil {
		log.Fatalf("unable to create crypt: %v", err)
	}

	reader, err := c.Decrypt(os.Stdin)
	if err != nil {
		log.Fatalf("unable to decrypt: %v", err)
	}

	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		log.Fatalf("unable to copy: %v", err)
	}
}
