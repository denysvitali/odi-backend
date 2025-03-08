package models

import (
	"fmt"
	"io"
	"time"
)

type ScannedPage struct {
	Reader     io.ReadSeeker
	ScanId     string
	SequenceId int
	ScanTime   time.Time
}

func (s ScannedPage) Id() string {
	return fmt.Sprintf("%s_%d", s.ScanId, s.SequenceId)
}
