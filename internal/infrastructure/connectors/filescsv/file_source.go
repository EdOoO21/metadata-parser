package filescsv

import (
	"context"
	"io"
)

type OpenedFile struct {
	Stream      io.ReadCloser
	Name        string
	Size        *int64
	ContentType string
}

type FileSource interface {
	CanHandle(ref FileReference) bool
	Exists(ctx context.Context, ref FileReference) (bool, error)
	OpenRead(ctx context.Context, ref FileReference) (*OpenedFile, error)
}
