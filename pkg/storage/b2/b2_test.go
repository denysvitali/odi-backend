package b2_test

import (
	"bytes"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"

	"github.com/denysvitali/odi-backend/pkg/models"
	"github.com/denysvitali/odi-backend/pkg/storage/b2"
)

func TestMain(m *testing.M) {
	logrus.StandardLogger().SetLevel(logrus.DebugLevel)
	os.Exit(m.Run())
}

var testEncryptionKey = "my key"

func TestB2_Store(t *testing.T) {
	if os.Getenv("E2E_TEST") != "true" {
		t.Skip("skipping test; E2E_TEST is not set")
	}
	b2Storage, err := b2.New(b2.Config{
		Account:    os.Getenv("B2_ACCOUNT"),
		Key:        os.Getenv("B2_KEY"),
		BucketName: os.Getenv("B2_BUCKET_NAME"),
	})
	if err != nil {
		t.Fatal(err)
	}

	err = b2Storage.Store(models.ScannedPage{
		Reader:     bytes.NewReader([]byte("hello world")),
		ScanId:     "test",
		SequenceId: 1,
		ScanTime:   time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestB2_StoreEncrypted(t *testing.T) {
	if os.Getenv("E2E_TEST") != "true" {
		t.Skip("skipping test; E2E_TEST is not set")
	}
	b2Storage, err := b2.New(b2.Config{
		Account:    os.Getenv("B2_ACCOUNT"),
		Key:        os.Getenv("B2_KEY"),
		BucketName: os.Getenv("B2_BUCKET_NAME"),
		Passphrase: testEncryptionKey,
	})
	if err != nil {
		t.Fatal(err)
	}

	err = b2Storage.Store(models.ScannedPage{
		Reader:     bytes.NewReader([]byte("hello world")),
		ScanId:     "test-encryption",
		SequenceId: 1,
		ScanTime:   time.Now(),
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestB2_RetrieveEncrypted(t *testing.T) {
	if os.Getenv("E2E_TEST") != "true" {
		t.Skip("skipping test; E2E_TEST is not set")
	}
	b2Storage, err := b2.New(b2.Config{
		Account:    os.Getenv("B2_ACCOUNT"),
		Key:        os.Getenv("B2_KEY"),
		BucketName: os.Getenv("B2_BUCKET_NAME"),
		Passphrase: testEncryptionKey,
	})
	if err != nil {
		t.Fatal(err)
	}

	s, err := b2Storage.Retrieve("test-encryption", 1)
	if err != nil {
		t.Fatal(err)
	}
	if s == nil {
		t.Fatal("s is nil")
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(s.Reader)
	if err != nil {
		t.Fatal(err)
	}

	if buf.String() != "hello world" {
		t.Fatalf("expected 'hello world', got '%s'", buf.String())
	}
}

func TestB2_Retrieve(t *testing.T) {
	if os.Getenv("E2E_TEST") != "true" {
		t.Skip("skipping test; E2E_TEST is not set")
	}
	b2Storage, err := b2.New(b2.Config{
		Account:    os.Getenv("B2_ACCOUNT"),
		Key:        os.Getenv("B2_KEY"),
		BucketName: os.Getenv("B2_BUCKET_NAME"),
	})
	if err != nil {
		t.Fatal(err)
	}

	s, err := b2Storage.Retrieve("test", 1)
	if err != nil {
		t.Fatal(err)
	}
	if s == nil {
		t.Fatal("s is nil")
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(s.Reader)
	if err != nil {
		t.Fatal(err)
	}

	if buf.String() != "hello world" {
		t.Fatalf("expected 'hello world', got '%s'", buf.String())
	}
}
