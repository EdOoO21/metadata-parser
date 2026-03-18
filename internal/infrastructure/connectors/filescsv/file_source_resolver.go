package filescsv

import "fmt"

type FileSourceResolver struct {
	sources []FileSource
}

func NewFileSourceResolver(sources ...FileSource) *FileSourceResolver {
	return &FileSourceResolver{sources: sources}
}

func (r *FileSourceResolver) Resolve(ref FileReference) (FileSource, error) {
	for _, source := range r.sources {
		if source.CanHandle(ref) {
			return source, nil
		}
	}

	return nil, fmt.Errorf("no file source registered for source=%q", ref.Source)
}
