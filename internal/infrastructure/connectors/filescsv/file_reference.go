package filescsv

import "strings"

type FileReference struct {
	Source      string
	Path        string
	DisplayName string
}

func LocalFile(path string) FileReference {
	return FileReference{
		Source: "local",
		Path:   path,
	}
}

func (r FileReference) SourceNormalized() string {
	return strings.ToLower(strings.TrimSpace(r.Source))
}
