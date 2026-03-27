package restopenapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/EdOoO21/metadata-parser/internal/application/contracts"
	"github.com/EdOoO21/metadata-parser/internal/domain/model"
	"github.com/EdOoO21/metadata-parser/internal/domain/types"
)

const defaultRESTTopValuesLimit = 5

func (s *Scanner) profileEndpoint(ctx context.Context, url string, dataset *contracts.ScannedDataset) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build endpoint request: %w", err)
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return fmt.Errorf("fetch endpoint response: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("fetch endpoint response: unexpected status %d", resp.StatusCode)
	}

	var payload any
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return fmt.Errorf("decode endpoint response: %w", err)
	}

	rows := responseRows(payload)
	rowCount := int64(len(rows))
	dataset.Dataset.RowCount = &rowCount

	profiles := make(map[string]*restColumnProfile, len(dataset.Columns))
	for i := range dataset.Columns {
		profiles[dataset.Columns[i].Column.Name] = newRESTColumnProfile(dataset.Columns[i].Column)
	}

	for _, row := range rows {
		flat := make(map[string]any)
		flattenRESTValue("", row, flat)
		for i := range dataset.Columns {
			profiles[dataset.Columns[i].Column.Name].Observe(flat[dataset.Columns[i].Column.Name])
		}
	}

	for i := range dataset.Columns {
		stat, topValues, err := profiles[dataset.Columns[i].Column.Name].Finalize()
		if err != nil {
			return fmt.Errorf("finalize column %q: %w", dataset.Columns[i].Column.Name, err)
		}
		dataset.Columns[i].Column.IsNullable = stat.NullCount > 0 || dataset.Columns[i].Column.IsNullable
		dataset.Columns[i].Stat = stat
		dataset.Columns[i].TopValues = topValues
	}

	dataset.Dataset.ProfileStatus = types.ProfileStatusProfiled
	dataset.Dataset.ProfileError = nil
	return nil
}

func responseRows(payload any) []any {
	switch typed := payload.(type) {
	case []any:
		return typed
	default:
		return []any{typed}
	}
}

func flattenRESTValue(prefix string, value any, out map[string]any) {
	switch typed := value.(type) {
	case map[string]any:
		for key, nested := range typed {
			name := key
			if prefix != "" {
				name = prefix + "." + key
			}
			flattenRESTValue(name, nested, out)
		}
	case []any:
		if prefix != "" {
			out[prefix] = typed
		}
	default:
		if prefix != "" {
			out[prefix] = typed
		}
	}
}

type restColumnProfile struct {
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

func newRESTColumnProfile(column model.Column) *restColumnProfile {
	return &restColumnProfile{
		column:      column,
		distinct:    make(map[string]struct{}),
		frequencies: make(map[string]int64),
	}
}

func (p *restColumnProfile) Observe(value any) {
	if value == nil {
		p.nullCount++
		return
	}

	valueJSON, err := json.Marshal(value)
	if err != nil {
		p.nullCount++
		return
	}

	p.nonNullCount++
	key := string(valueJSON)
	p.distinct[key] = struct{}{}
	p.frequencies[key]++
	p.observeMinMax(value)
}

func (p *restColumnProfile) observeMinMax(value any) {
	switch p.column.NormalizedType {
	case types.CanonicalTypeNumber:
		number, ok := restAsFloat64(value)
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
		timestamp, ok := restAsTimestamp(value)
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

func (p *restColumnProfile) Finalize() (*model.ColumnStat, []model.ColumnTopValue, error) {
	stat := &model.ColumnStat{
		NonNullCount:  p.nonNullCount,
		NullCount:     p.nullCount,
		DistinctCount: int64(len(p.distinct)),
	}

	if p.column.NormalizedType == types.CanonicalTypeNumber && p.hasNumber {
		minJSON, err := json.Marshal(p.minNumber)
		if err != nil {
			return nil, nil, err
		}
		maxJSON, err := json.Marshal(p.maxNumber)
		if err != nil {
			return nil, nil, err
		}
		stat.MinValueJSON = minJSON
		stat.MaxValueJSON = maxJSON
	}

	if p.column.NormalizedType == types.CanonicalTypeTimestamp && p.hasTime {
		minJSON, err := json.Marshal(p.minTime.Format(time.RFC3339Nano))
		if err != nil {
			return nil, nil, err
		}
		maxJSON, err := json.Marshal(p.maxTime.Format(time.RFC3339Nano))
		if err != nil {
			return nil, nil, err
		}
		stat.MinValueJSON = minJSON
		stat.MaxValueJSON = maxJSON
	}

	topValues := make([]model.ColumnTopValue, 0, len(p.frequencies))
	type item struct {
		valueJSON string
		count     int64
	}
	items := make([]item, 0, len(p.frequencies))
	for valueJSON, count := range p.frequencies {
		items = append(items, item{valueJSON: valueJSON, count: count})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].count == items[j].count {
			return items[i].valueJSON < items[j].valueJSON
		}
		return items[i].count > items[j].count
	})
	if len(items) > defaultRESTTopValuesLimit {
		items = items[:defaultRESTTopValuesLimit]
	}
	for i, item := range items {
		topValues = append(topValues, model.ColumnTopValue{
			Rank:            i + 1,
			ValueJSON:       []byte(item.valueJSON),
			OccurrenceCount: item.count,
		})
	}

	return stat, topValues, nil
}

func restAsFloat64(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case float32:
		return float64(typed), true
	case int:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case int32:
		return float64(typed), true
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

func restAsTimestamp(value any) (time.Time, bool) {
	text, ok := value.(string)
	if !ok {
		return time.Time{}, false
	}

	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02",
	} {
		parsed, err := time.Parse(layout, strings.TrimSpace(text))
		if err == nil {
			return parsed, true
		}
	}

	return time.Time{}, false
}
