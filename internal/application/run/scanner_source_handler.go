package run

import (
	"context"
	"fmt"

	"github.com/EdOoO21/metadata-parser/internal/application/contracts"
	appports "github.com/EdOoO21/metadata-parser/internal/application/ports"
	"github.com/EdOoO21/metadata-parser/internal/application/sensitive"
	"github.com/EdOoO21/metadata-parser/internal/domain/model"
	"github.com/EdOoO21/metadata-parser/internal/ports"
	"github.com/EdOoO21/metadata-parser/internal/settings"
)

type ScannerSourceHandler struct {
	logger ports.Logger
}

func NewScannerSourceHandler(logger ports.Logger) *ScannerSourceHandler {
	return &ScannerSourceHandler{
		logger: logger,
	}
}

func (h *ScannerSourceHandler) Handle(
	ctx context.Context,
	repo appports.CatalogRepository,
	runSourceID int64,
	src settings.SourceConfig,
	scanner appports.SourceScanner,
) error {
	if scanner == nil {
		return fmt.Errorf("source scanner is not configured")
	}

	result, err := scanner.ParseSource(ctx, src)
	if err != nil {
		return fmt.Errorf("parse source: %w", err)
	}

	if len(result.Datasets) == 0 {
		return fmt.Errorf("source %q did not return any datasets", src.Name)
	}

	if err := repo.WithTx(ctx, func(txRepo appports.CatalogRepository) error {
		for _, scannedDataset := range result.Datasets {
			if err := h.persistScannedDataset(ctx, txRepo, runSourceID, scannedDataset); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return fmt.Errorf("persist source %q datasets: %w", src.Name, err)
	}

	return nil
}

func (h *ScannerSourceHandler) persistScannedDataset(
	ctx context.Context,
	repo appports.CatalogRepository,
	runSourceID int64,
	scanned contracts.ScannedDataset,
) error {
	datasetToCreate := scanned.Dataset
	datasetToCreate.RunSourceID = runSourceID

	dataset, err := repo.CreateDataset(ctx, datasetToCreate)
	if err != nil {
		return fmt.Errorf("create dataset %q: %w", scanned.Dataset.Name, err)
	}

	for _, scannedColumn := range scanned.Columns {
		columnToCreate := scannedColumn.Column
		columnToCreate.DatasetID = dataset.ID

		column, err := repo.CreateColumn(ctx, columnToCreate)
		if err != nil {
			return fmt.Errorf("create column %q for dataset %q: %w", columnToCreate.Name, scanned.Dataset.Name, err)
		}

		if scannedColumn.Stat != nil {
			statToCreate := *scannedColumn.Stat
			statToCreate.ColumnID = column.ID

			stat, err := repo.CreateColumnStat(ctx, statToCreate)
			if err != nil {
				return fmt.Errorf("create stat for column %q: %w", columnToCreate.Name, err)
			}

			_, isSensitive := sensitive.MatchField(columnToCreate.Name, derefString(columnToCreate.Comment))
			if len(scannedColumn.TopValues) > 0 && !isSensitive {
				topValues := make([]model.ColumnTopValue, 0, len(scannedColumn.TopValues))
				for _, topValue := range scannedColumn.TopValues {
					topValue.ColumnStatID = stat.ID
					topValues = append(topValues, topValue)
				}

				if err := repo.CreateColumnTopValues(ctx, topValues); err != nil {
					return fmt.Errorf("create top values for column %q: %w", columnToCreate.Name, err)
				}
			}
		}
	}

	if h.logger != nil {
		h.logger.Info("dataset stored",
			"dataset_id", dataset.ID,
			"dataset_name", dataset.Name,
			"location", dataset.Location,
			"column_count", len(scanned.Columns),
		)
	}

	return nil
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
