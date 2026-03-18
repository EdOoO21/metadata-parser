package ports

import (
	"context"
	"time"

	"github.com/EdOoO21/metadata-parser/internal/domain/model"
	"github.com/EdOoO21/metadata-parser/internal/domain/types"
)

type CatalogRepository interface {
	WithTx(ctx context.Context, fn func(repo CatalogRepository) error) error

	EnsureSource(ctx context.Context, source model.Source) (*model.Source, error)

	CreateRun(ctx context.Context, run model.Run) (*model.Run, error)
	UpdateRunStatus(ctx context.Context, runID int64, status types.RunStatus, finishedAt *time.Time, errorMessage *string) error

	CreateRunSource(ctx context.Context, runSource model.RunSource) (*model.RunSource, error)
	UpdateRunSourceStatus(ctx context.Context, runSourceID int64, status types.RunStatus, finishedAt *time.Time, errorMessage *string) error

	CreateDataset(ctx context.Context, dataset model.Dataset) (*model.Dataset, error)
	CreateColumn(ctx context.Context, column model.Column) (*model.Column, error)
	CreateColumnStat(ctx context.Context, stat model.ColumnStat) (*model.ColumnStat, error)
	CreateColumnTopValues(ctx context.Context, values []model.ColumnTopValue) error
}
