package filescsv

import (
	"context"
	"encoding/csv"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

func normalizeCSVParseOptions(opts CSVParseOptions) CSVParseOptions {
	if opts.Delimiter == 0 {
		opts.Delimiter = ','
	}
	if strings.TrimSpace(opts.GeneratedColumnPrefix) == "" {
		opts.GeneratedColumnPrefix = "column_"
	}
	return opts
}

func readNextDataRecord(
	ctx context.Context,
	reader *csv.Reader,
	opts CSVParseOptions,
	recordCounter *int,
) ([]string, int, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, 0, ctx.Err()
		default:
		}

		record, err := reader.Read()
		if err != nil {
			return nil, 0, err
		}

		*recordCounter = *recordCounter + 1
		record = normalizeRecord(record, opts.TrimWhiteSpace)

		if opts.SkipEmptyRows && isEmptyRecord(record) {
			continue
		}

		return record, *recordCounter, nil
	}
}

func normalizeHeaders(headers []string, trim bool) []string {
	return normalizeRecord(headers, trim)
}

func normalizeRecord(record []string, trim bool) []string {
	out := make([]string, len(record))
	for i, value := range record {
		if trim {
			out[i] = strings.TrimSpace(value)
		} else {
			out[i] = value
		}
	}
	return out
}

func validateHeaders(headers []string) error {
	if len(headers) == 0 {
		return fmt.Errorf("csv must contain at least one header")
	}

	seen := make(map[string]struct{}, len(headers))
	for i, header := range headers {
		if strings.TrimSpace(header) == "" {
			return fmt.Errorf("csv header at position %d is empty", i)
		}
		key := strings.ToLower(header)
		if _, ok := seen[key]; ok {
			return fmt.Errorf("duplicate csv header detected: %q", header)
		}
		seen[key] = struct{}{}
	}
	return nil
}

func buildGeneratedHeaders(count int, prefix string) []string {
	headers := make([]string, count)
	for i := range count {
		headers[i] = fmt.Sprintf("%s%d", prefix, i+1)
	}
	return headers
}

func buildValues(headers, record []string) map[string]string {
	values := make(map[string]string, len(headers))
	for i := range headers {
		values[headers[i]] = record[i]
	}
	return values
}

func isEmptyRecord(record []string) bool {
	for _, value := range record {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

func trimUTF8BOM(s string) string {
	return strings.TrimPrefix(s, "\uFEFF")
}

func resolveLocalPath(path string) (string, error) {
	path = strings.TrimSpace(path)
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

func cloneStringMap(src map[string]string) map[string]string {
	dst := make(map[string]string, len(src))
	for key, value := range src {
		dst[key] = value
	}
	return dst
}

func discoverCSVPaths(path string, maxDepth int) ([]string, error) {
	rootPath, err := resolveLocalPath(path)
	if err != nil {
		return nil, err
	}

	info, err := os.Stat(rootPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("path not found: %s", path)
		}
		return nil, fmt.Errorf("stat path: %w", err)
	}

	if !info.IsDir() {
		if !isCSVPath(rootPath) {
			return nil, fmt.Errorf("unsupported file extension %q", strings.ToLower(filepath.Ext(rootPath)))
		}
		return []string{rootPath}, nil
	}

	var paths []string
	err = filepath.WalkDir(rootPath, func(currentPath string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(rootPath, currentPath)
		if err != nil {
			return err
		}

		depth := pathDepth(relPath)
		if d.IsDir() {
			if currentPath != rootPath && depth > maxDepth {
				return filepath.SkipDir
			}
			return nil
		}

		if depth > maxDepth {
			return nil
		}
		if isCSVPath(currentPath) {
			paths = append(paths, currentPath)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk directory: %w", err)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no csv files found under %s", rootPath)
	}

	return paths, nil
}

func pathDepth(relPath string) int {
	if relPath == "." || strings.TrimSpace(relPath) == "" {
		return 0
	}
	return strings.Count(filepath.Clean(relPath), string(filepath.Separator))
}

func isCSVPath(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".csv")
}
