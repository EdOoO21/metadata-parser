package report

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	appports "github.com/EdOoO21/metadata-parser/internal/application/ports"
	"github.com/EdOoO21/metadata-parser/internal/domain/model"
	"github.com/EdOoO21/metadata-parser/internal/domain/types"
)

type reportRepoStub struct {
	run  *model.Run
	rows []appports.ReportRow
	err  error
}

func (s *reportRepoStub) WithTx(ctx context.Context, fn func(repo appports.CatalogRepository) error) error {
	return fn(s)
}

func (s *reportRepoStub) EnsureSource(ctx context.Context, source model.Source) (*model.Source, error) {
	return nil, errors.New("not implemented")
}

func (s *reportRepoStub) CreateRun(ctx context.Context, run model.Run) (*model.Run, error) {
	return nil, errors.New("not implemented")
}

func (s *reportRepoStub) GetRun(ctx context.Context, runID int64) (*model.Run, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.run, nil
}

func (s *reportRepoStub) UpdateRunStatus(ctx context.Context, runID int64, status types.RunStatus, finishedAt *time.Time, errorMessage *string) error {
	return errors.New("not implemented")
}

func (s *reportRepoStub) CreateRunSource(ctx context.Context, runSource model.RunSource) (*model.RunSource, error) {
	return nil, errors.New("not implemented")
}

func (s *reportRepoStub) UpdateRunSourceStatus(ctx context.Context, runSourceID int64, status types.RunStatus, finishedAt *time.Time, errorMessage *string) error {
	return errors.New("not implemented")
}

func (s *reportRepoStub) CreateDataset(ctx context.Context, dataset model.Dataset) (*model.Dataset, error) {
	return nil, errors.New("not implemented")
}

func (s *reportRepoStub) CreateColumn(ctx context.Context, column model.Column) (*model.Column, error) {
	return nil, errors.New("not implemented")
}

func (s *reportRepoStub) CreateColumnStat(ctx context.Context, stat model.ColumnStat) (*model.ColumnStat, error) {
	return nil, errors.New("not implemented")
}

func (s *reportRepoStub) CreateColumnTopValues(ctx context.Context, values []model.ColumnTopValue) error {
	return errors.New("not implemented")
}

func (s *reportRepoStub) ListReportRows(ctx context.Context, runID int64) ([]appports.ReportRow, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.rows, nil
}

func TestReportCatalogUseCase_RendersMarkdownHTMLAndCSV(t *testing.T) {
	t.Parallel()

	rowCount := int64(2)
	run := &model.Run{
		ID:        42,
		StartedAt: time.Date(2026, 3, 26, 10, 0, 0, 0, time.UTC),
		Status:    types.RunStatusSuccess,
	}

	repo := &reportRepoStub{
		run: run,
		rows: []appports.ReportRow{
			{
				SourceName:           "demo_files",
				SourceKind:           types.SourceKindFiles,
				DatasetName:          "people.csv",
				DatasetKind:          types.DatasetKindFile,
				DatasetKey:           "/tmp/people.csv",
				DatasetLocation:      "/tmp/people.csv",
				DatasetRowCount:      &rowCount,
				DatasetProfileStatus: types.ProfileStatusProfiled,
				ColumnName:           "id",
				ColumnOriginalType:   "integer",
				ColumnNormalizedType: types.CanonicalTypeNumber,
				ColumnIsNullable:     false,
				ColumnOrdinal:        1,
			},
			{
				SourceName:           "demo_files",
				SourceKind:           types.SourceKindFiles,
				DatasetName:          "people.csv",
				DatasetKind:          types.DatasetKindFile,
				DatasetKey:           "/tmp/people.csv",
				DatasetLocation:      "/tmp/people.csv",
				DatasetRowCount:      &rowCount,
				DatasetProfileStatus: types.ProfileStatusProfiled,
				ColumnName:           "full_name",
				ColumnOriginalType:   "string",
				ColumnNormalizedType: types.CanonicalTypeString,
				ColumnIsNullable:     true,
				ColumnOrdinal:        2,
			},
		},
	}

	uc := NewReportCatalogUseCase()
	got, err := uc.Execute(context.Background(), ExecuteInput{
		Repository: repo,
		RunID:      42,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(got.Markdown, "# Catalog Report for Run 42") {
		t.Fatalf("markdown does not contain run header: %s", got.Markdown)
	}
	if !strings.Contains(got.Markdown, "## Source: demo_files (`files`)") {
		t.Fatalf("markdown does not contain source header: %s", got.Markdown)
	}
	if !strings.Contains(got.Markdown, "| 2 | full_name | string | STRING | yes |  |") {
		t.Fatalf("markdown does not contain column row: %s", got.Markdown)
	}
	if !strings.Contains(got.Markdown, "## Potentially Sensitive Fields") {
		t.Fatalf("markdown does not contain sensitive section: %s", got.Markdown)
	}
	if !strings.Contains(got.Markdown, "| demo_files | people.csv | full_name | person_name |") {
		t.Fatalf("markdown does not contain sensitive row: %s", got.Markdown)
	}
	if !strings.Contains(got.HTML, "<h1>Catalog Report for Run 42</h1>") {
		t.Fatalf("html does not contain run header: %s", got.HTML)
	}
	if !strings.Contains(got.HTML, "Source: demo_files") {
		t.Fatalf("html does not contain source header: %s", got.HTML)
	}
	if !strings.Contains(got.HTML, "<table>") {
		t.Fatalf("html does not contain table: %s", got.HTML)
	}
	if !strings.Contains(got.HTML, "<td>full_name</td>") {
		t.Fatalf("html does not contain column row: %s", got.HTML)
	}
	if !strings.Contains(got.HTML, "Potentially Sensitive Fields") {
		t.Fatalf("html does not contain sensitive section: %s", got.HTML)
	}
	if len(got.SensitiveFields) != 1 {
		t.Fatalf("expected one sensitive field, got %d", len(got.SensitiveFields))
	}

	csvOutput := string(got.CSV)
	if !strings.Contains(csvOutput, "source_name,source_kind,dataset_name") {
		t.Fatalf("csv does not contain header: %s", csvOutput)
	}
	if !strings.Contains(csvOutput, "demo_files,files,people.csv,file,/tmp/people.csv") {
		t.Fatalf("csv does not contain data row: %s", csvOutput)
	}
}

func TestReportCatalogUseCase_EmptyRunRendersMessage(t *testing.T) {
	t.Parallel()

	repo := &reportRepoStub{
		run: &model.Run{
			ID:        99,
			StartedAt: time.Date(2026, 3, 26, 10, 0, 0, 0, time.UTC),
			Status:    types.RunStatusSuccess,
		},
		rows: nil,
	}

	uc := NewReportCatalogUseCase()
	got, err := uc.Execute(context.Background(), ExecuteInput{
		Repository: repo,
		RunID:      99,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(got.Markdown, "Датасеты для этого запуска не найдены.") {
		t.Fatalf("unexpected markdown: %s", got.Markdown)
	}
	if !strings.Contains(got.HTML, "Датасеты для этого запуска не найдены.") {
		t.Fatalf("unexpected html: %s", got.HTML)
	}
	if len(got.SensitiveFields) != 0 {
		t.Fatalf("expected no sensitive fields, got %d", len(got.SensitiveFields))
	}
	if string(got.CSV) != "source_name,source_kind,dataset_name,dataset_kind,dataset_key,dataset_location,dataset_comment,dataset_row_count,dataset_profile_status,column_ordinal,column_name,column_original_type,column_normalized_type,column_is_nullable,column_comment\n" {
		t.Fatalf("unexpected csv output: %q", string(got.CSV))
	}
}
