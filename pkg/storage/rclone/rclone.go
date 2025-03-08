package rclone

import (
	"context"
	"time"

	"github.com/rclone/rclone/fs"
	"github.com/rclone/rclone/fs/hash"
)

type SourceFile struct {
	filePath string
	modTime  time.Time
	remote   string
	size     int64
}

func NewSourceFile(remote string, fileName string, modTime time.Time, fileSize int64) SourceFile {
	return SourceFile{
		filePath: fileName,
		modTime:  modTime,
		remote:   remote,
		size:     fileSize,
	}
}

type DummyFs struct{}

func (d DummyFs) Name() string {
	return "dummy"
}

func (d DummyFs) Root() string {
	return "/"
}

func (d DummyFs) String() string {
	return ""
}

func (d DummyFs) Precision() time.Duration {
	return time.Second
}

func (d DummyFs) Hashes() hash.Set {
	return hash.Set(hash.None)
}

func (d DummyFs) Features() *fs.Features {
	return &fs.Features{}
}

var _ fs.Info = (*DummyFs)(nil)

func (s SourceFile) String() string {
	return s.filePath
}

func (s SourceFile) Remote() string {
	return s.filePath
}

func (s SourceFile) ModTime(ctx context.Context) time.Time {
	return s.modTime
}

func (s SourceFile) Size() int64 {
	return s.size
}

func (s SourceFile) Fs() fs.Info {
	return DummyFs{}
}

func (s SourceFile) Hash(ctx context.Context, ty hash.Type) (string, error) {
	return "", nil
}

func (s SourceFile) Storable() bool {
	return true
}

var _ fs.ObjectInfo = (*SourceFile)(nil)
