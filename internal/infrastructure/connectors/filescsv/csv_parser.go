package filescsv

import (
	"bufio"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/EdOoO21/metadata-parser/internal/application/contracts"
	appports "github.com/EdOoO21/metadata-parser/internal/application/ports"
	"github.com/EdOoO21/metadata-parser/internal/domain/model"
	"github.com/EdOoO21/metadata-parser/internal/domain/types"
	"github.com/EdOoO21/metadata-parser/internal/settings"
)

const defaultCSVSampleRows = 10

type CSVParseOptions struct {
	HasHeaderRecord       bool
	Delimiter             rune
	TrimWhiteSpace        bool
	SkipEmptyRows         bool
	GeneratedColumnPrefix string
	MaxRows               int
}

func DefaultCSVParseOptions() CSVParseOptions {
	return CSVParseOptions{
		HasHeaderRecord:       true,
		Delimiter:             ',',
		TrimWhiteSpace:        true,
		SkipEmptyRows:         true,
		GeneratedColumnPrefix: "column_",
		MaxRows:               0,
	}
}

type CSVRow struct {
	RecordNumber int
	Values       map[string]string
}

type CSVParseResult struct {
	Headers []string
	Rows    []CSVRow
}

type CSVParser struct{}

func NewCSVParser() *CSVParser {
	return &CSVParser{}
}

func (p *CSVParser) ParseSource(ctx context.Context, src settings.SourceConfig) (*contracts.SourceScanResult, error) {
	opts := DefaultCSVParseOptions()
	opts.MaxRows = defaultCSVSampleRows

	effectiveConfigJSON, err := json.Marshal(src.Config)
	if err != nil {
		return nil, fmt.Errorf("marshal effective config: %w", err)
	}

	paths, err := discoverCSVPaths(src.Config.Path, src.Config.MaxDepth)
	if err != nil {
		return nil, err
	}

	scannedDatasets := make([]contracts.ScannedDataset, 0, len(paths))
	profiler := NewCSVProfiler()
	for _, path := range paths {
		rowCount, err := p.CountRows(ctx, path, opts)
		if err != nil {
			return nil, fmt.Errorf("count csv rows %q: %w", path, err)
		}

		result, err := p.ParseFile(ctx, path, opts)
		if err != nil {
			return nil, fmt.Errorf("parse csv file %q: %w", path, err)
		}

		dataset, err := profiler.BuildDataset(path, result, rowCount, opts.MaxRows)
		if err != nil {
			return nil, fmt.Errorf("build scanned dataset %q: %w", path, err)
		}

		scannedDatasets = append(scannedDatasets, dataset)
	}

	return &contracts.SourceScanResult{
		Source: model.Source{
			Name: src.Name,
			Kind: types.SourceKind(src.Kind),
		},
		EffectiveConfigJSON: effectiveConfigJSON,
		Datasets:            scannedDatasets,
	}, nil
}

func (p *CSVParser) CountRows(ctx context.Context, path string, opts CSVParseOptions) (int64, error) {
	resolvedPath, err := resolveLocalPath(path)
	if err != nil {
		return 0, err
	}

	file, err := os.Open(resolvedPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, fmt.Errorf("file not found: %s", path)
		}
		return 0, fmt.Errorf("open local file: %w", err)
	}
	defer file.Close()

	return p.Count(ctx, file, opts)
}

func (p *CSVParser) ParseFile(ctx context.Context, path string, opts CSVParseOptions) (*CSVParseResult, error) {
	resolvedPath, err := resolveLocalPath(path)
	if err != nil {
		return nil, err
	}

	file, err := os.Open(resolvedPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, fmt.Errorf("file not found: %s", path)
		}
		return nil, fmt.Errorf("open local file: %w", err)
	}
	defer file.Close()

	return p.Parse(ctx, file, opts)
}

func (p *CSVParser) Count(ctx context.Context, r io.Reader, opts CSVParseOptions) (int64, error) {
	opts = normalizeCSVParseOptions(opts)

	reader := csv.NewReader(bufio.NewReader(r))
	reader.Comma = opts.Delimiter
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = opts.TrimWhiteSpace
	reader.ReuseRecord = false

	recordCounter := 0
	rowCount := int64(0)

	if opts.HasHeaderRecord {
		_, _, err := readNextDataRecord(ctx, reader, opts, &recordCounter)
		if err != nil {
			if err == io.EOF {
				return 0, nil
			}
			return 0, err
		}
	}

	for {
		_, _, err := readNextDataRecord(ctx, reader, opts, &recordCounter)
		if err != nil {
			if err == io.EOF {
				break
			}
			return 0, err
		}
		rowCount++
	}

	return rowCount, nil
}

func (p *CSVParser) Parse(ctx context.Context, r io.Reader, opts CSVParseOptions) (*CSVParseResult, error) {
	opts = normalizeCSVParseOptions(opts)

	reader := csv.NewReader(bufio.NewReader(r))
	reader.Comma = opts.Delimiter
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = opts.TrimWhiteSpace
	reader.ReuseRecord = false

	recordCounter := 0
	rows := make([]CSVRow, 0, 32)

	firstRecord, firstRecordNumber, err := readNextDataRecord(ctx, reader, opts, &recordCounter)
	if err != nil {
		if err == io.EOF {
			return &CSVParseResult{Headers: []string{}, Rows: []CSVRow{}}, nil
		}
		return nil, err
	}

	if len(firstRecord) > 0 {
		firstRecord[0] = trimUTF8BOM(firstRecord[0])
	}

	var headers []string
	if opts.HasHeaderRecord {
		headers = normalizeHeaders(firstRecord, opts.TrimWhiteSpace)
		if err := validateHeaders(headers); err != nil {
			return nil, err
		}
	} else {
		headers = buildGeneratedHeaders(len(firstRecord), opts.GeneratedColumnPrefix)
		rows = append(rows, CSVRow{
			RecordNumber: firstRecordNumber,
			Values:       buildValues(headers, firstRecord),
		})
		if opts.MaxRows > 0 && len(rows) >= opts.MaxRows {
			return &CSVParseResult{Headers: headers, Rows: rows}, nil
		}
	}

	for {
		record, recordNumber, err := readNextDataRecord(ctx, reader, opts, &recordCounter)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		if len(record) != len(headers) {
			return nil, fmt.Errorf("csv record %d has %d field(s), expected %d", recordNumber, len(record), len(headers))
		}

		rows = append(rows, CSVRow{
			RecordNumber: recordNumber,
			Values:       buildValues(headers, record),
		})

		if opts.MaxRows > 0 && len(rows) >= opts.MaxRows {
			break
		}
	}

	return &CSVParseResult{Headers: headers, Rows: rows}, nil
}

var _ appports.SourceScanner = (*CSVParser)(nil)
