package diff

import (
	"context"
	"fmt"
	"sort"
	"strings"

	appports "github.com/EdOoO21/metadata-parser/internal/application/ports"
)

type ExecuteInput struct {
	Repository appports.CatalogRepository
	FromRunID  int64
	ToRunID    int64
}

type DiffCatalogUseCase struct{}

func NewDiffCatalogUseCase() *DiffCatalogUseCase {
	return &DiffCatalogUseCase{}
}

func (uc *DiffCatalogUseCase) Execute(ctx context.Context, input ExecuteInput) (string, error) {
	if input.Repository == nil {
		return "", fmt.Errorf("catalog repository is not configured")
	}
	if input.FromRunID <= 0 || input.ToRunID <= 0 {
		return "", fmt.Errorf("from and to run ids must be greater than zero")
	}

	if _, err := input.Repository.GetRun(ctx, input.FromRunID); err != nil {
		return "", fmt.Errorf("get from run: %w", err)
	}
	if _, err := input.Repository.GetRun(ctx, input.ToRunID); err != nil {
		return "", fmt.Errorf("get to run: %w", err)
	}

	fromRows, err := input.Repository.ListReportRows(ctx, input.FromRunID)
	if err != nil {
		return "", fmt.Errorf("list from report rows: %w", err)
	}
	toRows, err := input.Repository.ListReportRows(ctx, input.ToRunID)
	if err != nil {
		return "", fmt.Errorf("list to report rows: %w", err)
	}

	return renderDiff(fromRows, toRows, input.FromRunID, input.ToRunID), nil
}

type datasetSnapshot struct {
	label   string
	columns map[string]columnSnapshot
}

type columnSnapshot struct {
	name           string
	originalType   string
	normalizedType string
	isNullable     bool
	comment        string
}

func renderDiff(fromRows, toRows []appports.ReportRow, fromRunID, toRunID int64) string {
	fromDatasets := buildDatasetSnapshots(fromRows)
	toDatasets := buildDatasetSnapshots(toRows)

	addedDatasets := diffAddedDatasets(fromDatasets, toDatasets)
	removedDatasets := diffAddedDatasets(toDatasets, fromDatasets)
	addedColumns := diffAddedColumns(fromDatasets, toDatasets)
	removedColumns := diffAddedColumns(toDatasets, fromDatasets)
	changedColumns := diffChangedColumns(fromDatasets, toDatasets)

	var b strings.Builder
	b.WriteString(fmt.Sprintf("# Catalog Diff: Run %d -> Run %d\n\n", fromRunID, toRunID))

	writeSection(&b, "Added Datasets", addedDatasets)
	writeSection(&b, "Removed Datasets", removedDatasets)
	writeSection(&b, "Added Columns", addedColumns)
	writeSection(&b, "Removed Columns", removedColumns)
	writeSection(&b, "Changed Columns", changedColumns)

	if len(addedDatasets) == 0 && len(removedDatasets) == 0 && len(addedColumns) == 0 && len(removedColumns) == 0 && len(changedColumns) == 0 {
		b.WriteString("Изменений между выбранными слепками не найдено.\n")
	}

	return b.String()
}

func buildDatasetSnapshots(rows []appports.ReportRow) map[string]datasetSnapshot {
	result := make(map[string]datasetSnapshot, len(rows))

	for _, row := range rows {
		datasetKey := row.SourceName + "::" + row.DatasetKey
		snapshot, ok := result[datasetKey]
		if !ok {
			snapshot = datasetSnapshot{
				label:   fmt.Sprintf("%s / %s", row.SourceName, row.DatasetName),
				columns: map[string]columnSnapshot{},
			}
		}

		snapshot.columns[row.ColumnName] = columnSnapshot{
			name:           row.ColumnName,
			originalType:   row.ColumnOriginalType,
			normalizedType: string(row.ColumnNormalizedType),
			isNullable:     row.ColumnIsNullable,
			comment:        derefString(row.ColumnComment),
		}

		result[datasetKey] = snapshot
	}

	return result
}

func diffAddedDatasets(base, target map[string]datasetSnapshot) []string {
	items := make([]string, 0)

	for key, dataset := range target {
		if _, ok := base[key]; ok {
			continue
		}
		items = append(items, dataset.label)
	}

	sort.Strings(items)
	return items
}

func diffAddedColumns(base, target map[string]datasetSnapshot) []string {
	items := make([]string, 0)

	for datasetKey, targetDataset := range target {
		baseDataset, ok := base[datasetKey]
		if !ok {
			continue
		}

		for columnName := range targetDataset.columns {
			if _, exists := baseDataset.columns[columnName]; exists {
				continue
			}
			items = append(items, fmt.Sprintf("%s: %s", targetDataset.label, columnName))
		}
	}

	sort.Strings(items)
	return items
}

func diffChangedColumns(fromDatasets, toDatasets map[string]datasetSnapshot) []string {
	items := make([]string, 0)

	for datasetKey, fromDataset := range fromDatasets {
		toDataset, ok := toDatasets[datasetKey]
		if !ok {
			continue
		}

		for columnName, fromColumn := range fromDataset.columns {
			toColumn, ok := toDataset.columns[columnName]
			if !ok {
				continue
			}

			changes := make([]string, 0, 3)
			if fromColumn.normalizedType != toColumn.normalizedType || fromColumn.originalType != toColumn.originalType {
				changes = append(changes, fmt.Sprintf(
					"type %s/%s -> %s/%s",
					fromColumn.originalType,
					fromColumn.normalizedType,
					toColumn.originalType,
					toColumn.normalizedType,
				))
			}
			if fromColumn.isNullable != toColumn.isNullable {
				changes = append(changes, fmt.Sprintf("nullable %t -> %t", fromColumn.isNullable, toColumn.isNullable))
			}
			if fromColumn.comment != toColumn.comment {
				changes = append(changes, fmt.Sprintf("comment %q -> %q", fromColumn.comment, toColumn.comment))
			}

			if len(changes) == 0 {
				continue
			}

			items = append(items, fmt.Sprintf("%s: %s (%s)", toDataset.label, columnName, strings.Join(changes, ", ")))
		}
	}

	sort.Strings(items)
	return items
}

func writeSection(b *strings.Builder, title string, items []string) {
	b.WriteString(fmt.Sprintf("## %s\n", title))
	if len(items) == 0 {
		b.WriteString("- none\n\n")
		return
	}

	for _, item := range items {
		b.WriteString("- ")
		b.WriteString(item)
		b.WriteString("\n")
	}
	b.WriteString("\n")
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
