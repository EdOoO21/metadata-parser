package filescsv

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type LocalFileSource struct{}

func NewLocalFileSource() *LocalFileSource {
	return &LocalFileSource{}
}

func (s *LocalFileSource) CanHandle(ref FileReference) bool {
	switch ref.SourceNormalized() {
	case "", "local", "file":
		return true
	default:
		return false
	}
}

func (s *LocalFileSource) Exists(_ context.Context, ref FileReference) (bool, error) {
	path, err := resolveLocalPath(ref)
	if err != nil {
		return false, err
	}

	_, err = os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}

	return false, fmt.Errorf("stat local file: %w", err)
}

func (s *LocalFileSource) OpenRead(_ context.Context, ref FileReference) (*OpenedFile, error) {
	path, err := resolveLocalPath(ref)
	if err != nil {
		return nil, err
	}

	stat, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("stat local file: %w", err)
	}

	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open local file: %w", err)
	}

	size := stat.Size()
	name := strings.TrimSpace(ref.DisplayName)
	if name == "" {
		name = stat.Name()
	}

	return &OpenedFile{
		Stream:      file,
		Name:        name,
		Size:        &size,
		ContentType: detectContentType(path),
	}, nil
}

func resolveLocalPath(ref FileReference) (string, error) {
	path := strings.TrimSpace(ref.Path)
	if path == "" {
		return "", errors.New("file path is required")
	}

	if strings.HasPrefix(strings.ToLower(path), "file://") {
		u, err := url.Parse(path)
		if err != nil {
			return "", fmt.Errorf("parse file url: %w", err)
		}
		path = u.Path
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path: %w", err)
	}

	return abs, nil
}

func detectContentType(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".csv":
		return "text/csv"
	case ".txt":
		return "text/plain"
	default:
		return ""
	}
}
