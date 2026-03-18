package filescsv

import (
	"context"
	"fmt"
)

type PreviewCSVService struct {
	resolver     *FileSourceResolver
	csvConnector CSVConnector
}

func NewPreviewCSVService(resolver *FileSourceResolver, csvConnector CSVConnector) *PreviewCSVService {
	return &PreviewCSVService{
		resolver:     resolver,
		csvConnector: csvConnector,
	}
}

func (s *PreviewCSVService) Execute(ctx context.Context, ref FileReference, maxRows int) (*CSVReadResult, error) {
	source, err := s.resolver.Resolve(ref)
	if err != nil {
		return nil, err
	}

	exists, err := source.Exists(ctx, ref)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("file not found: %s", ref.Path)
	}

	opened, err := source.OpenRead(ctx, ref)
	if err != nil {
		return nil, err
	}
	defer opened.Stream.Close()

	opts := DefaultCSVReadOptions()
	opts.MaxRows = maxRows

	return s.csvConnector.Read(ctx, opened.Stream, opts)
}
