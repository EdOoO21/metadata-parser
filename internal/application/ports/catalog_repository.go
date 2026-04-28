package ports

import (
	"context"
	"time"

	"github.com/EdOoO21/metadata-parser/internal/domain/model"
	"github.com/EdOoO21/metadata-parser/internal/domain/types"
)

type ReportRow struct {
	SourceName           string
	SourceKind           types.SourceKind
	DatasetName          string
	DatasetKind          types.DatasetKind
	DatasetKey           string
	DatasetLocation      string
	DatasetComment       *string
	DatasetRowCount      *int64
	DatasetProfileStatus types.ProfileStatus
	DatasetMetadataJSON  []byte
	ColumnPresent        bool
	ColumnName           string
	ColumnOriginalType   string
	ColumnNormalizedType types.CanonicalType
	ColumnIsNullable     bool
	ColumnComment        *string
	ColumnOrdinal        int
}

type CatalogRepository interface {
	// WithTx выполняет набор операций в одной транзакции каталога.
	WithTx(ctx context.Context, fn func(repo CatalogRepository) error) error

	// EnsureSource находит или создает источник по его идентичности.
	EnsureSource(ctx context.Context, source model.Source) (*model.Source, error)

	// CreateRun создает верхнеуровневый слепок запуска.
	CreateRun(ctx context.Context, run model.Run) (*model.Run, error)
	// GetRun читает сохраненный run по идентификатору для отчетов и diff.
	GetRun(ctx context.Context, runID int64) (*model.Run, error)
	// UpdateRunStatus завершает run итоговым статусом и ошибкой при наличии.
	UpdateRunStatus(ctx context.Context, runID int64, status types.RunStatus, finishedAt *time.Time, errorMessage *string) error

	// CreateRunSource создает запись о запуске конкретного источника внутри run.
	CreateRunSource(ctx context.Context, runSource model.RunSource) (*model.RunSource, error)
	// UpdateRunSourceStatus завершает обработку конкретного источника.
	UpdateRunSourceStatus(ctx context.Context, runSourceID int64, status types.RunStatus, finishedAt *time.Time, errorMessage *string) error

	// CreateDataset сохраняет найденный датасет в составе слепка источника.
	CreateDataset(ctx context.Context, dataset model.Dataset) (*model.Dataset, error)
	// CreateColumn сохраняет описание колонки датасета.
	CreateColumn(ctx context.Context, column model.Column) (*model.Column, error)
	// CreateColumnStat сохраняет агрегированную статистику по колонке.
	CreateColumnStat(ctx context.Context, stat model.ColumnStat) (*model.ColumnStat, error)
	// CreateColumnTopValues сохраняет top-N популярных значений по колонке.
	CreateColumnTopValues(ctx context.Context, values []model.ColumnTopValue) error
	// ListReportRows читает плоское представление run для генерации карты таблиц и CSV-экспорта.
	ListReportRows(ctx context.Context, runID int64) ([]ReportRow, error)
}
