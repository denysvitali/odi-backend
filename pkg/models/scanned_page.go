package models

import (
	"fmt"
	"io"
	"time"
)

type ScannedPage struct {
	Reader     io.ReadSeeker
	ScanID     string
	SequenceID int
	ScanTime   time.Time
}

func (s ScannedPage) ID() string {
	return fmt.Sprintf("%s_%d", s.ScanID, s.SequenceID)
}
