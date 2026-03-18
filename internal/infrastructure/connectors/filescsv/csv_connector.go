package filescsv

import (
	"bufio"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

type CSVReadOptions struct {
	HasHeaderRecord       bool
	Delimiter             rune
	TrimWhiteSpace        bool
	SkipEmptyRows         bool
	GeneratedColumnPrefix string
	MaxRows               int
}

func DefaultCSVReadOptions() CSVReadOptions {
	return CSVReadOptions{
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

type CSVReadResult struct {
	Headers []string
	Rows    []CSVRow
}

type CSVConnector interface {
	Read(ctx context.Context, r io.Reader, opts CSVReadOptions) (*CSVReadResult, error)
}

type DefaultCSVConnector struct{}

func NewCSVConnector() *DefaultCSVConnector {
	return &DefaultCSVConnector{}
}

func (c *DefaultCSVConnector) Read(ctx context.Context, r io.Reader, opts CSVReadOptions) (*CSVReadResult, error) {
	opts = normalizeCSVReadOptions(opts)

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
			return &CSVReadResult{Headers: []string{}, Rows: []CSVRow{}}, nil
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
			return &CSVReadResult{Headers: headers, Rows: rows}, nil
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

	return &CSVReadResult{Headers: headers, Rows: rows}, nil
}

func normalizeCSVReadOptions(opts CSVReadOptions) CSVReadOptions {
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
	opts CSVReadOptions,
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
	for i := 0; i < count; i++ {
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
