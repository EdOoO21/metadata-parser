package parquet

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/EdOoO21/metadata-parser/internal/application/contracts"
	"github.com/EdOoO21/metadata-parser/internal/domain/model"
	"github.com/EdOoO21/metadata-parser/internal/domain/types"
	pq "github.com/parquet-go/parquet-go"
)

const defaultParquetTopValuesLimit = 5

type topValueItem struct {
	valueJSON string
	count     int64
}

func profileParquetColumns(file io.ReaderAt, columns []contracts.ScannedColumn) error {
	reader := pq.NewGenericReader[any](file)
	defer reader.Close()

	profiles := make(map[string]*parquetColumnProfile, len(columns))
	for i := range columns {
		profiles[columns[i].Column.Name] = newParquetColumnProfile(columns[i].Column)
	}

	rows := make([]any, 64)
	for {
		n, err := reader.Read(rows)
		if n > 0 {
			for _, row := range rows[:n] {
				values := make(map[string]any)
				flattenParquetRow("", row, values)
				for i := range columns {
					profiles[columns[i].Column.Name].Observe(values[columns[i].Column.Name])
				}
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("read parquet rows: %w", err)
		}
	}

	for i := range columns {
		stat, topValues, err := profiles[columns[i].Column.Name].Finalize()
		if err != nil {
			return fmt.Errorf("finalize column %q: %w", columns[i].Column.Name, err)
		}
		columns[i].Column.IsNullable = stat.NullCount > 0
		columns[i].Stat = stat
		columns[i].TopValues = topValues
	}

	return nil
}

func flattenParquetRow(prefix string, value any, out map[string]any) {
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			name := key
			if prefix != "" {
				name = prefix + "." + key
			}
			flattenParquetRow(name, nested, out)
		}
	case []any:
		if prefix != "" {
			out[prefix] = typed
		}
		grouped := make(map[string][]any)
		for _, item := range typed {
			object, ok := item.(map[string]any)
			if !ok {
				continue
			}
			for key, nested := range object {
				grouped[key] = append(grouped[key], nested)
			}
		}
		for key, groupedValues := range grouped {
			name := key
			if prefix != "" {
				name = prefix + "." + key
			}
			out[name] = groupedValues
		}
	default:
		if prefix != "" {
			out[prefix] = typed
		}
	}
}

type parquetColumnProfile struct {
	column       model.Column
	nonNullCount int64
	nullCount    int64
	distinct     map[string]struct{}
	frequencies  map[string]int64
	minNumber    float64
	maxNumber    float64
	hasNumber    bool
	minTime      time.Time
	maxTime      time.Time
	hasTime      bool
}

func newParquetColumnProfile(column model.Column) *parquetColumnProfile {
	return &parquetColumnProfile{
		column:      column,
		distinct:    make(map[string]struct{}),
		frequencies: make(map[string]int64),
	}
}

func (p *parquetColumnProfile) Observe(value any) {
	if value == nil {
		p.nullCount++
		return
	}

	p.nonNullCount++

	key, valueJSON, ok := encodeParquetValue(value)
	if !ok {
		p.nullCount++
		p.nonNullCount--
		return
	}

	p.distinct[key] = struct{}{}
	p.frequencies[string(valueJSON)]++
	p.observeMinMax(value)
}

func (p *parquetColumnProfile) observeMinMax(value any) {
	switch p.column.NormalizedType {
	case types.CanonicalTypeNumber:
		number, ok := asFloat64(value)
		if !ok {
			return
		}
		if !p.hasNumber {
			p.minNumber = number
			p.maxNumber = number
			p.hasNumber = true
			return
		}
		if number < p.minNumber {
			p.minNumber = number
		}
		if number > p.maxNumber {
			p.maxNumber = number
		}
	case types.CanonicalTypeTimestamp:
		timestamp, ok := asTimestamp(value)
		if !ok {
			return
		}
		if !p.hasTime {
			p.minTime = timestamp
			p.maxTime = timestamp
			p.hasTime = true
			return
		}
		if timestamp.Before(p.minTime) {
			p.minTime = timestamp
		}
		if timestamp.After(p.maxTime) {
			p.maxTime = timestamp
		}
	}
}

func (p *parquetColumnProfile) Finalize() (*model.ColumnStat, []model.ColumnTopValue, error) {
	stat := &model.ColumnStat{
		NonNullCount:  p.nonNullCount,
		NullCount:     p.nullCount,
		DistinctCount: int64(len(p.distinct)),
	}

	switch p.column.NormalizedType {
	case types.CanonicalTypeNumber:
		if p.hasNumber {
			minValue, maxValue, err := marshalParquetNumberBounds(p.minNumber, p.maxNumber, p.column.OriginalType)
			if err != nil {
				return nil, nil, err
			}
			stat.MinValueJSON = minValue
			stat.MaxValueJSON = maxValue
		}
	case types.CanonicalTypeTimestamp:
		if p.hasTime {
			minValue, err := json.Marshal(p.minTime.Format(time.RFC3339Nano))
			if err != nil {
				return nil, nil, err
			}
			maxValue, err := json.Marshal(p.maxTime.Format(time.RFC3339Nano))
			if err != nil {
				return nil, nil, err
			}
			stat.MinValueJSON = minValue
			stat.MaxValueJSON = maxValue
		}
	}

	topValues, err := buildParquetTopValues(p.frequencies)
	if err != nil {
		return nil, nil, err
	}

	return stat, topValues, nil
}

func encodeParquetValue(value any) (string, []byte, bool) {
	valueJSON, err := json.Marshal(value)
	if err != nil {
		return "", nil, false
	}
	return string(valueJSON), valueJSON, true
}

func buildParquetTopValues(frequencies map[string]int64) ([]model.ColumnTopValue, error) {
	if len(frequencies) == 0 {
		return nil, nil
	}

	items := make([]topValueItem, 0, len(frequencies))
	for valueJSON, count := range frequencies {
		items = append(items, topValueItem{valueJSON: valueJSON, count: count})
	}

	sortParquetTopValues(items)

	limit := len(items)
	if limit > defaultParquetTopValuesLimit {
		limit = defaultParquetTopValuesLimit
	}

	topValues := make([]model.ColumnTopValue, 0, limit)
	for i := 0; i < limit; i++ {
		topValues = append(topValues, model.ColumnTopValue{
			Rank:            i + 1,
			ValueJSON:       []byte(items[i].valueJSON),
			OccurrenceCount: items[i].count,
		})
	}

	return topValues, nil
}

func sortParquetTopValues(items []topValueItem) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].count == items[j].count {
			return items[i].valueJSON < items[j].valueJSON
		}
		return items[i].count > items[j].count
	})
}

func marshalParquetNumberBounds(minValue float64, maxValue float64, originalType string) ([]byte, []byte, error) {
	if looksIntegerType(originalType) {
		minJSON, err := json.Marshal(int64(minValue))
		if err != nil {
			return nil, nil, err
		}
		maxJSON, err := json.Marshal(int64(maxValue))
		if err != nil {
			return nil, nil, err
		}
		return minJSON, maxJSON, nil
	}

	minJSON, err := json.Marshal(minValue)
	if err != nil {
		return nil, nil, err
	}
	maxJSON, err := json.Marshal(maxValue)
	if err != nil {
		return nil, nil, err
	}
	return minJSON, maxJSON, nil
}

func looksIntegerType(originalType string) bool {
	value := strings.ToLower(strings.TrimSpace(originalType))
	return strings.Contains(value, "int") || strings.Contains(value, "uint")
}

func asFloat64(value any) (float64, bool) {
	switch typed := value.(type) {
	case int:
		return float64(typed), true
	case int8:
		return float64(typed), true
	case int16:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case uint:
		return float64(typed), true
	case uint8:
		return float64(typed), true
	case uint16:
		return float64(typed), true
	case uint32:
		return float64(typed), true
	case uint64:
		return float64(typed), true
	case float32:
		return float64(typed), true
	case float64:
		return typed, true
	case json.Number:
		parsed, err := typed.Float64()
		return parsed, err == nil
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func asTimestamp(value any) (time.Time, bool) {
	switch typed := value.(type) {
	case time.Time:
		return typed, true
	case string:
		layouts := []string{
			time.RFC3339Nano,
			time.RFC3339,
			"2006-01-02 15:04:05",
			"2006-01-02",
		}
		for _, layout := range layouts {
			parsed, err := time.Parse(layout, strings.TrimSpace(typed))
			if err == nil {
				return parsed, true
			}
		}
	}
	return time.Time{}, false
}
