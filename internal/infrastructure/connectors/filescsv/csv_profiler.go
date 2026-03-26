package filescsv

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/EdOoO21/metadata-parser/internal/application/contracts"
	"github.com/EdOoO21/metadata-parser/internal/domain/model"
	"github.com/EdOoO21/metadata-parser/internal/domain/types"
)

const defaultTopValuesLimit = 5

type CSVProfiler struct{}

func NewCSVProfiler() *CSVProfiler {
	return &CSVProfiler{}
}

func (p *CSVProfiler) BuildDataset(path string, result *CSVParseResult, sampleLimit int) (contracts.ScannedDataset, error) {
	metadataJSON, err := json.Marshal(map[string]any{
		"format":       "csv",
		"headers":      append([]string(nil), result.Headers...),
		"sample_rows":  minInt(len(result.Rows), sampleLimit),
		"sample_limit": sampleLimit,
		"source_path":  path,
	})
	if err != nil {
		return contracts.ScannedDataset{}, err
	}

	datasetName := filepath.Base(path)
	if strings.TrimSpace(datasetName) == "" {
		datasetName = path
	}

	rowCount := int64(len(result.Rows))
	columns, err := p.profileColumns(result.Headers, result.Rows)
	if err != nil {
		return contracts.ScannedDataset{}, err
	}

	return contracts.ScannedDataset{
		Dataset: model.Dataset{
			Kind:          types.DatasetKindFile,
			DatasetKey:    path,
			Name:          datasetName,
			Location:      path,
			RowCount:      &rowCount,
			DiscoveredAt:  time.Now().UTC(),
			ProfileStatus: types.ProfileStatusProfiled,
			MetadataJSON:  metadataJSON,
		},
		Columns: columns,
	}, nil
}

func (p *CSVProfiler) profileColumns(headers []string, rows []CSVRow) ([]contracts.ScannedColumn, error) {
	columns := make([]contracts.ScannedColumn, 0, len(headers))

	for i, header := range headers {
		profile, err := profileColumn(header, i+1, rows)
		if err != nil {
			return nil, fmt.Errorf("profile column %q: %w", header, err)
		}
		columns = append(columns, profile)
	}

	return columns, nil
}

func profileColumn(header string, ordinal int, rows []CSVRow) (contracts.ScannedColumn, error) {
	values := make([]string, 0, len(rows))
	nonNullCount := int64(0)
	nullCount := int64(0)
	distinct := make(map[string]struct{})
	frequencies := make(map[string]int64)

	typeState := detectTypeState{}

	for _, row := range rows {
		rawValue := row.Values[header]
		if isNullishCSVValue(rawValue) {
			nullCount++
			continue
		}

		value := strings.TrimSpace(rawValue)
		values = append(values, value)
		nonNullCount++
		distinct[value] = struct{}{}
		frequencies[value]++
		typeState.Observe(value)
	}

	column := model.Column{
		Name:            header,
		OrdinalPosition: ordinal,
		IsNullable:      nullCount > 0,
	}

	if nonNullCount == 0 {
		column.OriginalType = "string"
		column.NormalizedType = types.CanonicalTypeString

		return contracts.ScannedColumn{
			Column: column,
			Stat: &model.ColumnStat{
				NonNullCount:  0,
				NullCount:     nullCount,
				DistinctCount: 0,
			},
		}, nil
	}

	column.OriginalType, column.NormalizedType = finalizeTypes(typeState)

	stat := model.ColumnStat{
		NonNullCount:  nonNullCount,
		NullCount:     nullCount,
		DistinctCount: int64(len(distinct)),
	}

	minJSON, maxJSON, err := buildMinMaxJSON(values, column.NormalizedType, typeState)
	if err != nil {
		return contracts.ScannedColumn{}, err
	}
	stat.MinValueJSON = minJSON
	stat.MaxValueJSON = maxJSON

	topValues, err := buildTopValues(frequencies, column.NormalizedType)
	if err != nil {
		return contracts.ScannedColumn{}, err
	}

	return contracts.ScannedColumn{
		Column:    column,
		Stat:      &stat,
		TopValues: topValues,
	}, nil
}

type detectTypeState struct {
	allInt       bool
	allNumber    bool
	allBoolean   bool
	allTimestamp bool
}

func (s *detectTypeState) Observe(value string) {
	if !s.initialized() {
		s.allInt = true
		s.allNumber = true
		s.allBoolean = true
		s.allTimestamp = true
	}

	if _, err := strconv.ParseInt(value, 10, 64); err != nil {
		s.allInt = false
	}
	if _, err := strconv.ParseFloat(value, 64); err != nil {
		s.allNumber = false
	}
	if !isBooleanValue(value) {
		s.allBoolean = false
	}
	if _, err := parseTimestamp(value); err != nil {
		s.allTimestamp = false
	}
}

func (s detectTypeState) initialized() bool {
	return s.allInt || s.allNumber || s.allBoolean || s.allTimestamp
}

func finalizeTypes(state detectTypeState) (string, types.CanonicalType) {
	switch {
	case state.allBoolean:
		return "boolean", types.CanonicalTypeBoolean
	case state.allInt:
		return "integer", types.CanonicalTypeNumber
	case state.allNumber:
		return "number", types.CanonicalTypeNumber
	case state.allTimestamp:
		return "timestamp", types.CanonicalTypeTimestamp
	default:
		return "string", types.CanonicalTypeString
	}
}

func buildMinMaxJSON(values []string, canonicalType types.CanonicalType, state detectTypeState) ([]byte, []byte, error) {
	switch canonicalType {
	case types.CanonicalTypeNumber:
		if state.allInt {
			minValue, maxValue, err := minMaxInt(values)
			if err != nil {
				return nil, nil, err
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

		minValue, maxValue, err := minMaxFloat(values)
		if err != nil {
			return nil, nil, err
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

	case types.CanonicalTypeTimestamp:
		minValue, maxValue, err := minMaxTimestamp(values)
		if err != nil {
			return nil, nil, err
		}
		minJSON, err := json.Marshal(minValue.Format(time.RFC3339Nano))
		if err != nil {
			return nil, nil, err
		}
		maxJSON, err := json.Marshal(maxValue.Format(time.RFC3339Nano))
		if err != nil {
			return nil, nil, err
		}
		return minJSON, maxJSON, nil

	default:
		return nil, nil, nil
	}
}

func buildTopValues(frequencies map[string]int64, canonicalType types.CanonicalType) ([]model.ColumnTopValue, error) {
	if len(frequencies) == 0 {
		return nil, nil
	}

	type topValueItem struct {
		value string
		count int64
	}

	items := make([]topValueItem, 0, len(frequencies))
	for value, count := range frequencies {
		items = append(items, topValueItem{value: value, count: count})
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].count == items[j].count {
			return items[i].value < items[j].value
		}
		return items[i].count > items[j].count
	})

	limit := minInt(len(items), defaultTopValuesLimit)
	topValues := make([]model.ColumnTopValue, 0, limit)
	for i := 0; i < limit; i++ {
		valueJSON, err := parseJSONValue(items[i].value, canonicalType)
		if err != nil {
			return nil, err
		}
		topValues = append(topValues, model.ColumnTopValue{
			Rank:            i + 1,
			ValueJSON:       valueJSON,
			OccurrenceCount: items[i].count,
		})
	}

	return topValues, nil
}

func minMaxInt(values []string) (int64, int64, error) {
	minValue, err := strconv.ParseInt(values[0], 10, 64)
	if err != nil {
		return 0, 0, err
	}
	maxValue := minValue

	for _, value := range values[1:] {
		parsed, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return 0, 0, err
		}
		if parsed < minValue {
			minValue = parsed
		}
		if parsed > maxValue {
			maxValue = parsed
		}
	}

	return minValue, maxValue, nil
}

func minMaxFloat(values []string) (float64, float64, error) {
	minValue, err := strconv.ParseFloat(values[0], 64)
	if err != nil {
		return 0, 0, err
	}
	maxValue := minValue

	for _, value := range values[1:] {
		parsed, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return 0, 0, err
		}
		if parsed < minValue {
			minValue = parsed
		}
		if parsed > maxValue {
			maxValue = parsed
		}
	}

	return minValue, maxValue, nil
}

func minMaxTimestamp(values []string) (time.Time, time.Time, error) {
	minValue, err := parseTimestamp(values[0])
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	maxValue := minValue

	for _, value := range values[1:] {
		parsed, err := parseTimestamp(value)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		if parsed.Before(minValue) {
			minValue = parsed
		}
		if parsed.After(maxValue) {
			maxValue = parsed
		}
	}

	return minValue, maxValue, nil
}
