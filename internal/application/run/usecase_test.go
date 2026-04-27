package run

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/EdOoO21/metadata-parser/internal/application/contracts"
	appports "github.com/EdOoO21/metadata-parser/internal/application/ports"
	"github.com/EdOoO21/metadata-parser/internal/domain/model"
	"github.com/EdOoO21/metadata-parser/internal/domain/types"
	"github.com/EdOoO21/metadata-parser/internal/settings"
)

type loggerStub struct{}

func (l *loggerStub) Info(string, ...any)  {}
func (l *loggerStub) Warn(string, ...any)  {}
func (l *loggerStub) Error(string, ...any) {}

type sourceScannerStub struct {
	result *contracts.SourceScanResult
	err    error
}

func (s *sourceScannerStub) ParseSource(ctx context.Context, src settings.SourceConfig) (*contracts.SourceScanResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.result, nil
}

type sourceScannerFactoryStub struct {
	scanner appports.SourceScanner
	err     error
}

func (f *sourceScannerFactoryStub) ForSource(src settings.SourceConfig) (appports.SourceScanner, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.scanner, nil
}

type catalogRepoStub struct {
	nextSourceID       int64
	nextRunID          int64
	nextRunSourceID    int64
	nextDatasetID      int64
	nextColumnID       int64
	nextColumnStatID   int64
	withTxCalls        int
	runStatusCalls     []types.RunStatus
	runStatusErrors    []*string
	runSourceStatuses  []types.RunStatus
	runSourceErrors    []*string
	createdDatasets    []model.Dataset
	createdColumns     []model.Column
	createdStats       []model.ColumnStat
	createdTopValues   [][]model.ColumnTopValue
	createRunErr       error
	updateRunErr       error
	ensureSourceErr    error
	withTxErr          error
	createRunSourceErr error
	updateRunSrcErr    error
	createDatasetErr   error
	createColumnErr    error
	createStatErr      error
	createTopValuesErr error
}

func (r *catalogRepoStub) WithTx(ctx context.Context, fn func(repo appports.CatalogRepository) error) error {
	r.withTxCalls++
	if r.withTxErr != nil {
		return r.withTxErr
	}
	return fn(r)
}

func (r *catalogRepoStub) EnsureSource(ctx context.Context, source model.Source) (*model.Source, error) {
	if r.ensureSourceErr != nil {
		return nil, r.ensureSourceErr
	}
	r.nextSourceID++
	source.ID = r.nextSourceID
	return &source, nil
}

func (r *catalogRepoStub) CreateRun(ctx context.Context, run model.Run) (*model.Run, error) {
	if r.createRunErr != nil {
		return nil, r.createRunErr
	}
	if r.nextRunID == 0 {
		r.nextRunID = 1
	}
	run.ID = r.nextRunID
	return &run, nil
}

func (r *catalogRepoStub) GetRun(ctx context.Context, runID int64) (*model.Run, error) {
	return nil, errors.New("unexpected test call")
}

func (r *catalogRepoStub) UpdateRunStatus(ctx context.Context, runID int64, status types.RunStatus, finishedAt *time.Time, errorMessage *string) error {
	if r.updateRunErr != nil {
		return r.updateRunErr
	}
	r.runStatusCalls = append(r.runStatusCalls, status)
	r.runStatusErrors = append(r.runStatusErrors, errorMessage)
	return nil
}

func (r *catalogRepoStub) CreateRunSource(ctx context.Context, runSource model.RunSource) (*model.RunSource, error) {
	if r.createRunSourceErr != nil {
		return nil, r.createRunSourceErr
	}
	r.nextRunSourceID++
	runSource.ID = r.nextRunSourceID
	return &runSource, nil
}

func (r *catalogRepoStub) UpdateRunSourceStatus(ctx context.Context, runSourceID int64, status types.RunStatus, finishedAt *time.Time, errorMessage *string) error {
	if r.updateRunSrcErr != nil {
		return r.updateRunSrcErr
	}
	r.runSourceStatuses = append(r.runSourceStatuses, status)
	r.runSourceErrors = append(r.runSourceErrors, errorMessage)
	return nil
}

func (r *catalogRepoStub) CreateDataset(ctx context.Context, dataset model.Dataset) (*model.Dataset, error) {
	if r.createDatasetErr != nil {
		return nil, r.createDatasetErr
	}
	r.nextDatasetID++
	dataset.ID = r.nextDatasetID
	r.createdDatasets = append(r.createdDatasets, dataset)
	return &dataset, nil
}

func (r *catalogRepoStub) CreateColumn(ctx context.Context, column model.Column) (*model.Column, error) {
	if r.createColumnErr != nil {
		return nil, r.createColumnErr
	}
	r.nextColumnID++
	column.ID = r.nextColumnID
	r.createdColumns = append(r.createdColumns, column)
	return &column, nil
}

func (r *catalogRepoStub) CreateColumnStat(ctx context.Context, stat model.ColumnStat) (*model.ColumnStat, error) {
	if r.createStatErr != nil {
		return nil, r.createStatErr
	}
	r.nextColumnStatID++
	stat.ID = r.nextColumnStatID
	r.createdStats = append(r.createdStats, stat)
	return &stat, nil
}

func (r *catalogRepoStub) CreateColumnTopValues(ctx context.Context, values []model.ColumnTopValue) error {
	if r.createTopValuesErr != nil {
		return r.createTopValuesErr
	}
	cloned := make([]model.ColumnTopValue, len(values))
	copy(cloned, values)
	r.createdTopValues = append(r.createdTopValues, cloned)
	return nil
}

func (r *catalogRepoStub) ListReportRows(ctx context.Context, runID int64) ([]appports.ReportRow, error) {
	return nil, errors.New("unexpected test call")
}

func TestDetermineRunStatus(t *testing.T) {
	t.Parallel()

	if got := determineRunStatus(2, 0); got != types.RunStatusSuccess {
		t.Fatalf("expected success, got %s", got)
	}
	if got := determineRunStatus(0, 2); got != types.RunStatusFailed {
		t.Fatalf("expected failed, got %s", got)
	}
	if got := determineRunStatus(1, 1); got != types.RunStatusPartial {
		t.Fatalf("expected partial, got %s", got)
	}
}

func TestScannerSourceHandler_HandlePersistsDatasetsColumnsStatsAndTopValues(t *testing.T) {
	t.Parallel()

	repo := &catalogRepoStub{}
	handler := NewScannerSourceHandler(&loggerStub{})
	rowCount := int64(2)
	stat := &model.ColumnStat{NonNullCount: 2, DistinctCount: 2}

	scanner := &sourceScannerStub{
		result: &contracts.SourceScanResult{
			Datasets: []contracts.ScannedDataset{
				{
					Dataset: model.Dataset{
						Name:          "people.csv",
						DatasetKey:    "people.csv",
						Location:      "/tmp/people.csv",
						RowCount:      &rowCount,
						ProfileStatus: types.ProfileStatusProfiled,
					},
					Columns: []contracts.ScannedColumn{
						{
							Column: model.Column{
								Name:            "full_name",
								OrdinalPosition: 1,
							},
							Stat: stat,
							TopValues: []model.ColumnTopValue{
								{Rank: 1, ValueJSON: []byte(`"Ivan Ivanov"`), OccurrenceCount: 1},
							},
						},
					},
				},
			},
		},
	}

	err := handler.Handle(context.Background(), repo, 77, settings.SourceConfig{Name: "files_demo"}, scanner)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(repo.createdDatasets) != 1 || repo.createdDatasets[0].RunSourceID != 77 {
		t.Fatalf("unexpected created datasets: %+v", repo.createdDatasets)
	}
	if len(repo.createdColumns) != 1 || repo.createdColumns[0].DatasetID == 0 {
		t.Fatalf("unexpected created columns: %+v", repo.createdColumns)
	}
	if len(repo.createdStats) != 1 || repo.createdStats[0].ColumnID == 0 {
		t.Fatalf("unexpected created stats: %+v", repo.createdStats)
	}
	if len(repo.createdTopValues) != 1 || repo.createdTopValues[0][0].ColumnStatID == 0 {
		t.Fatalf("unexpected top values: %+v", repo.createdTopValues)
	}
	if repo.withTxCalls != 1 {
		t.Fatalf("expected one transaction, got %d", repo.withTxCalls)
	}
}

func TestSourceProcessor_ProcessMarksSourceFailedOnScannerError(t *testing.T) {
	t.Parallel()

	repo := &catalogRepoStub{}
	handler := NewScannerSourceHandler(&loggerStub{})
	processor := NewSourceProcessor(
		&loggerStub{},
		&sourceScannerFactoryStub{scanner: &sourceScannerStub{err: errors.New("boom")}},
		handler,
	)

	err := processor.Process(context.Background(), repo, 10, settings.SourceConfig{
		Name: "broken_source",
		Kind: "files",
	})
	if err == nil {
		t.Fatal("expected error")
	}

	if len(repo.runSourceStatuses) != 1 || repo.runSourceStatuses[0] != types.RunStatusFailed {
		t.Fatalf("unexpected run source statuses: %+v", repo.runSourceStatuses)
	}
	if repo.runSourceErrors[0] == nil || *repo.runSourceErrors[0] == "" {
		t.Fatalf("expected run source error message, got %+v", repo.runSourceErrors)
	}
}

func TestRunCatalogUseCase_ExecuteReturnsPartialOnMixedSources(t *testing.T) {
	t.Parallel()

	repo := &catalogRepoStub{}
	successScanner := &sourceScannerStub{
		result: &contracts.SourceScanResult{
			Datasets: []contracts.ScannedDataset{
				{
					Dataset: model.Dataset{
						Name:          "people.csv",
						DatasetKey:    "people.csv",
						Location:      "/tmp/people.csv",
						ProfileStatus: types.ProfileStatusDiscoveredOnly,
					},
					Columns: []contracts.ScannedColumn{
						{Column: model.Column{Name: "id", OrdinalPosition: 1}},
					},
				},
			},
		},
	}
	factory := &sequenceScannerFactory{
		scanners: []appports.SourceScanner{
			successScanner,
			&sourceScannerStub{err: errors.New("bad source")},
		},
	}

	uc := NewRunCatalogUseCase(
		&loggerStub{},
		NewSourceProcessor(&loggerStub{}, factory, NewScannerSourceHandler(&loggerStub{})),
	)

	cfg := &settings.AppConfig{
		Version: 1,
		Catalog: settings.CatalogConfig{DSNEnv: "CATALOG_DSN"},
		Sources: []settings.SourceConfig{
			{Name: "ok_source", Kind: "files"},
			{Name: "bad_source", Kind: "files"},
		},
	}

	runID, err := uc.Execute(context.Background(), ExecuteInput{
		Repository:         repo,
		Config:             cfg,
		ConfigHash:         "hash",
		ConfigSnapshotJSON: []byte(`{}`),
	})
	if err == nil {
		t.Fatal("expected partial error")
	}
	if runID != 1 {
		t.Fatalf("expected run id 1, got %d", runID)
	}
	var partialErr *CompletedWithErrorsError
	if !errors.As(err, &partialErr) {
		t.Fatalf("expected CompletedWithErrorsError, got %T: %v", err, err)
	}
	if partialErr.RunID != 1 {
		t.Fatalf("expected partial error run id 1, got %d", partialErr.RunID)
	}
	if len(repo.runStatusCalls) != 1 || repo.runStatusCalls[0] != types.RunStatusPartial {
		t.Fatalf("unexpected run statuses: %+v", repo.runStatusCalls)
	}
	if repo.runStatusErrors[0] == nil || *repo.runStatusErrors[0] == "" {
		t.Fatalf("expected aggregated run error, got %+v", repo.runStatusErrors)
	}
}

type sequenceScannerFactory struct {
	scanners []appports.SourceScanner
	index    int
}

func (f *sequenceScannerFactory) ForSource(src settings.SourceConfig) (appports.SourceScanner, error) {
	if f.index >= len(f.scanners) {
		return nil, errors.New("no scanner configured")
	}
	scanner := f.scanners[f.index]
	f.index++
	return scanner, nil
}

func TestScannerSourceHandler_HandleErrors(t *testing.T) {
	t.Parallel()

	repo := &catalogRepoStub{}
	handler := NewScannerSourceHandler(&loggerStub{})

	if err := handler.Handle(context.Background(), repo, 1, settings.SourceConfig{Name: "x"}, nil); err == nil {
		t.Fatal("expected nil scanner error")
	}

	err := handler.Handle(context.Background(), repo, 1, settings.SourceConfig{Name: "x"}, &sourceScannerStub{
		result: &contracts.SourceScanResult{},
	})
	if err == nil {
		t.Fatal("expected no dataset error")
	}
}

func TestScannerSourceHandler_PersistErrors(t *testing.T) {
	t.Parallel()

	baseResult := &contracts.SourceScanResult{
		Datasets: []contracts.ScannedDataset{
			{
				Dataset: model.Dataset{Name: "people"},
				Columns: []contracts.ScannedColumn{
					{
						Column: model.Column{Name: "id"},
						Stat:   &model.ColumnStat{},
						TopValues: []model.ColumnTopValue{
							{Rank: 1, ValueJSON: []byte(`1`), OccurrenceCount: 1},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name string
		repo *catalogRepoStub
	}{
		{name: "dataset", repo: &catalogRepoStub{createDatasetErr: errors.New("dataset fail")}},
		{name: "column", repo: &catalogRepoStub{createColumnErr: errors.New("column fail")}},
		{name: "stat", repo: &catalogRepoStub{createStatErr: errors.New("stat fail")}},
		{name: "top_values", repo: &catalogRepoStub{createTopValuesErr: errors.New("top fail")}},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			handler := NewScannerSourceHandler(&loggerStub{})
			err := handler.Handle(context.Background(), tt.repo, 5, settings.SourceConfig{Name: "files"}, &sourceScannerStub{result: baseResult})
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestScannerSourceHandler_HandleReturnsTransactionError(t *testing.T) {
	t.Parallel()

	repo := &catalogRepoStub{withTxErr: errors.New("tx fail")}
	handler := NewScannerSourceHandler(&loggerStub{})

	err := handler.Handle(context.Background(), repo, 5, settings.SourceConfig{Name: "files"}, &sourceScannerStub{
		result: &contracts.SourceScanResult{
			Datasets: []contracts.ScannedDataset{
				{Dataset: model.Dataset{Name: "people"}},
			},
		},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "persist source \"files\" datasets: tx fail") {
		t.Fatalf("unexpected error: %v", err)
	}
	if repo.withTxCalls != 1 {
		t.Fatalf("expected one transaction call, got %d", repo.withTxCalls)
	}
}

func TestSourceProcessor_HandleByKindDependencyErrors(t *testing.T) {
	t.Parallel()

	processor := NewSourceProcessor(&loggerStub{}, nil, nil)
	err := processor.handleByKind(context.Background(), &catalogRepoStub{}, 1, settings.SourceConfig{Name: "x", Kind: "files"})
	if err == nil || !strings.Contains(err.Error(), "source scanner factory") {
		t.Fatalf("unexpected error: %v", err)
	}

	processor = NewSourceProcessor(&loggerStub{}, &sourceScannerFactoryStub{}, nil)
	err = processor.handleByKind(context.Background(), &catalogRepoStub{}, 1, settings.SourceConfig{Name: "x", Kind: "files"})
	if err == nil || !strings.Contains(err.Error(), "source handler") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSourceProcessor_ProcessSuccessAndRepositoryErrors(t *testing.T) {
	t.Parallel()

	successScanner := &sourceScannerStub{
		result: &contracts.SourceScanResult{
			Datasets: []contracts.ScannedDataset{
				{
					Dataset: model.Dataset{Name: "people"},
					Columns: []contracts.ScannedColumn{{Column: model.Column{Name: "id"}}},
				},
			},
		},
	}

	processor := NewSourceProcessor(
		&loggerStub{},
		&sourceScannerFactoryStub{scanner: successScanner},
		NewScannerSourceHandler(&loggerStub{}),
	)

	repo := &catalogRepoStub{}
	if err := processor.Process(context.Background(), repo, 1, settings.SourceConfig{Name: "ok", Kind: "files"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(repo.runSourceStatuses) != 1 || repo.runSourceStatuses[0] != types.RunStatusSuccess {
		t.Fatalf("unexpected statuses: %+v", repo.runSourceStatuses)
	}

	for _, repoWithErr := range []*catalogRepoStub{
		{ensureSourceErr: errors.New("ensure fail")},
		{createRunSourceErr: errors.New("run source fail")},
		{updateRunSrcErr: errors.New("update fail")},
	} {
		err := processor.Process(context.Background(), repoWithErr, 1, settings.SourceConfig{Name: "bad", Kind: "files"})
		if err == nil {
			t.Fatal("expected error")
		}
	}
}

func TestRunCatalogUseCase_ExecuteCreateRunAndUpdateErrorsAndSuccess(t *testing.T) {
	t.Parallel()

	cfg := &settings.AppConfig{
		Version: 1,
		Catalog: settings.CatalogConfig{DSNEnv: "CATALOG_DSN"},
		Sources: []settings.SourceConfig{
			{Name: "ok_source", Kind: "files"},
		},
	}

	successScanner := &sourceScannerStub{
		result: &contracts.SourceScanResult{
			Datasets: []contracts.ScannedDataset{
				{
					Dataset: model.Dataset{Name: "people"},
					Columns: []contracts.ScannedColumn{{Column: model.Column{Name: "id"}}},
				},
			},
		},
	}

	newUseCase := func(repo *catalogRepoStub) *RunCatalogUseCase {
		return NewRunCatalogUseCase(
			&loggerStub{},
			NewSourceProcessor(
				&loggerStub{},
				&sourceScannerFactoryStub{scanner: successScanner},
				NewScannerSourceHandler(&loggerStub{}),
			),
		)
	}

	if _, err := newUseCase(&catalogRepoStub{createRunErr: errors.New("create fail")}).Execute(context.Background(), ExecuteInput{
		Repository:         &catalogRepoStub{createRunErr: errors.New("create fail")},
		Config:             cfg,
		ConfigHash:         "hash",
		ConfigSnapshotJSON: []byte(`{}`),
	}); err == nil {
		t.Fatal("expected create run error")
	}

	repoUpdateFail := &catalogRepoStub{updateRunErr: errors.New("update fail")}
	if _, err := newUseCase(repoUpdateFail).Execute(context.Background(), ExecuteInput{
		Repository:         repoUpdateFail,
		Config:             cfg,
		ConfigHash:         "hash",
		ConfigSnapshotJSON: []byte(`{}`),
	}); err == nil {
		t.Fatal("expected update run error")
	}

	repoSuccess := &catalogRepoStub{}
	runID, err := newUseCase(repoSuccess).Execute(context.Background(), ExecuteInput{
		Repository:         repoSuccess,
		Config:             cfg,
		ConfigHash:         "hash",
		ConfigSnapshotJSON: []byte(`{}`),
	})
	if err != nil {
		t.Fatalf("unexpected success error: %v", err)
	}
	if runID != 1 {
		t.Fatalf("unexpected run id: %d", runID)
	}
	if len(repoSuccess.runStatusCalls) != 1 || repoSuccess.runStatusCalls[0] != types.RunStatusSuccess {
		t.Fatalf("unexpected success statuses: %+v", repoSuccess.runStatusCalls)
	}
	if repoSuccess.runStatusErrors[0] != nil {
		t.Fatalf("unexpected error message: %+v", repoSuccess.runStatusErrors[0])
	}
}
