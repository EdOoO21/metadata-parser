package diff

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

type diffRepoStub struct {
	rowsByRun map[int64][]appports.ReportRow
	runsByID  map[int64]*model.Run
	err       error
}

func (s *diffRepoStub) WithTx(ctx context.Context, fn func(repo appports.CatalogRepository) error) error {
	return fn(s)
}

func (s *diffRepoStub) EnsureSource(ctx context.Context, source model.Source) (*model.Source, error) {
	return nil, errors.New("not implemented")
}

func (s *diffRepoStub) CreateRun(ctx context.Context, run model.Run) (*model.Run, error) {
	return nil, errors.New("not implemented")
}

func (s *diffRepoStub) GetRun(ctx context.Context, runID int64) (*model.Run, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.runsByID[runID], nil
}

func (s *diffRepoStub) UpdateRunStatus(ctx context.Context, runID int64, status types.RunStatus, finishedAt *time.Time, errorMessage *string) error {
	return errors.New("not implemented")
}

func (s *diffRepoStub) CreateRunSource(ctx context.Context, runSource model.RunSource) (*model.RunSource, error) {
	return nil, errors.New("not implemented")
}

func (s *diffRepoStub) UpdateRunSourceStatus(ctx context.Context, runSourceID int64, status types.RunStatus, finishedAt *time.Time, errorMessage *string) error {
	return errors.New("not implemented")
}

func (s *diffRepoStub) CreateDataset(ctx context.Context, dataset model.Dataset) (*model.Dataset, error) {
	return nil, errors.New("not implemented")
}

func (s *diffRepoStub) CreateColumn(ctx context.Context, column model.Column) (*model.Column, error) {
	return nil, errors.New("not implemented")
}

func (s *diffRepoStub) CreateColumnStat(ctx context.Context, stat model.ColumnStat) (*model.ColumnStat, error) {
	return nil, errors.New("not implemented")
}

func (s *diffRepoStub) CreateColumnTopValues(ctx context.Context, values []model.ColumnTopValue) error {
	return errors.New("not implemented")
}

func (s *diffRepoStub) ListReportRows(ctx context.Context, runID int64) ([]appports.ReportRow, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.rowsByRun[runID], nil
}

func TestDiffCatalogUseCase_RendersChanges(t *testing.T) {
	t.Parallel()

	oldComment := "old comment"
	newComment := "new comment"

	repo := &diffRepoStub{
		runsByID: map[int64]*model.Run{
			41: {ID: 41, Status: types.RunStatusSuccess},
			42: {ID: 42, Status: types.RunStatusSuccess},
		},
		rowsByRun: map[int64][]appports.ReportRow{
			41: {
				{
					SourceName:           "demo_files",
					DatasetName:          "people.csv",
					DatasetKey:           "/tmp/people.csv",
					ColumnName:           "id",
					ColumnOriginalType:   "integer",
					ColumnNormalizedType: types.CanonicalTypeNumber,
					ColumnIsNullable:     false,
				},
				{
					SourceName:           "demo_files",
					DatasetName:          "people.csv",
					DatasetKey:           "/tmp/people.csv",
					ColumnName:           "name",
					ColumnOriginalType:   "string",
					ColumnNormalizedType: types.CanonicalTypeString,
					ColumnIsNullable:     true,
					ColumnComment:        &oldComment,
				},
				{
					SourceName:           "demo_files",
					DatasetName:          "orders.csv",
					DatasetKey:           "/tmp/orders.csv",
					ColumnName:           "amount",
					ColumnOriginalType:   "number",
					ColumnNormalizedType: types.CanonicalTypeNumber,
					ColumnIsNullable:     false,
				},
			},
			42: {
				{
					SourceName:           "demo_files",
					DatasetName:          "people.csv",
					DatasetKey:           "/tmp/people.csv",
					ColumnName:           "id",
					ColumnOriginalType:   "integer",
					ColumnNormalizedType: types.CanonicalTypeNumber,
					ColumnIsNullable:     false,
				},
				{
					SourceName:           "demo_files",
					DatasetName:          "people.csv",
					DatasetKey:           "/tmp/people.csv",
					ColumnName:           "name",
					ColumnOriginalType:   "text",
					ColumnNormalizedType: types.CanonicalTypeString,
					ColumnIsNullable:     false,
					ColumnComment:        &newComment,
				},
				{
					SourceName:           "demo_files",
					DatasetName:          "people.csv",
					DatasetKey:           "/tmp/people.csv",
					ColumnName:           "email",
					ColumnOriginalType:   "string",
					ColumnNormalizedType: types.CanonicalTypeString,
					ColumnIsNullable:     true,
				},
				{
					SourceName:           "demo_files",
					DatasetName:          "cities.csv",
					DatasetKey:           "/tmp/cities.csv",
					ColumnName:           "city",
					ColumnOriginalType:   "string",
					ColumnNormalizedType: types.CanonicalTypeString,
					ColumnIsNullable:     false,
				},
			},
		},
	}

	uc := NewDiffCatalogUseCase()
	got, err := uc.Execute(context.Background(), ExecuteInput{
		Repository: repo,
		FromRunID:  41,
		ToRunID:    42,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(got, "# Catalog Diff: Run 41 -> Run 42") {
		t.Fatalf("missing diff header: %s", got)
	}
	if !strings.Contains(got, "- demo_files / cities.csv") {
		t.Fatalf("missing added dataset: %s", got)
	}
	if !strings.Contains(got, "- demo_files / orders.csv") {
		t.Fatalf("missing removed dataset: %s", got)
	}
	if !strings.Contains(got, "- demo_files / people.csv: email") {
		t.Fatalf("missing added column: %s", got)
	}
	if !strings.Contains(got, "- demo_files / people.csv: name (type string/STRING -> text/STRING, nullable true -> false, comment \"old comment\" -> \"new comment\")") {
		t.Fatalf("missing changed column: %s", got)
	}
}

func TestDiffCatalogUseCase_NoChanges(t *testing.T) {
	t.Parallel()

	rows := []appports.ReportRow{
		{
			SourceName:           "demo_files",
			DatasetName:          "people.csv",
			DatasetKey:           "/tmp/people.csv",
			ColumnName:           "id",
			ColumnOriginalType:   "integer",
			ColumnNormalizedType: types.CanonicalTypeNumber,
			ColumnIsNullable:     false,
		},
	}

	repo := &diffRepoStub{
		runsByID: map[int64]*model.Run{
			1: {ID: 1, Status: types.RunStatusSuccess},
			2: {ID: 2, Status: types.RunStatusSuccess},
		},
		rowsByRun: map[int64][]appports.ReportRow{
			1: rows,
			2: rows,
		},
	}

	uc := NewDiffCatalogUseCase()
	got, err := uc.Execute(context.Background(), ExecuteInput{
		Repository: repo,
		FromRunID:  1,
		ToRunID:    2,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(got, "Изменений между выбранными слепками не найдено.") {
		t.Fatalf("unexpected diff output: %s", got)
	}
}
