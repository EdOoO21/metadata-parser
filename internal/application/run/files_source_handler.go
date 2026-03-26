package run

import (
	"context"
	"fmt"

	"github.com/EdOoO21/metadata-parser/internal/application/contracts"
	appports "github.com/EdOoO21/metadata-parser/internal/application/ports"
	"github.com/EdOoO21/metadata-parser/internal/domain/model"
	"github.com/EdOoO21/metadata-parser/internal/ports"
	"github.com/EdOoO21/metadata-parser/internal/settings"
)

type FilesSourceHandler struct {
	csvParser appports.CSVParser
	logger    ports.Logger
}

func NewFilesSourceHandler(csvParser appports.CSVParser, logger ports.Logger) *FilesSourceHandler {
	return &FilesSourceHandler{
		csvParser: csvParser,
		logger:    logger,
	}
}

func (h *FilesSourceHandler) Handle(
	ctx context.Context,
	repo appports.CatalogRepository,
	runSourceID int64,
	src settings.SourceConfig,
) error {
	if h.csvParser == nil {
		return fmt.Errorf("csv parser is not configured")
	}

	result, err := h.csvParser.ParseSource(ctx, src)
	if err != nil {
		return fmt.Errorf("parse csv source: %w", err)
	}

	if len(result.Datasets) == 0 {
		return fmt.Errorf("files source %q did not return any datasets", src.Name)
	}

	for _, scannedDataset := range result.Datasets {
		if err := h.persistScannedDataset(ctx, repo, runSourceID, scannedDataset); err != nil {
			return err
		}
	}

	return nil
}

func (h *FilesSourceHandler) persistScannedDataset(
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

			if len(scannedColumn.TopValues) > 0 {
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
		h.logger.Info("file dataset stored",
			"dataset_id", dataset.ID,
			"dataset_name", dataset.Name,
			"location", dataset.Location,
			"column_count", len(scanned.Columns),
		)
	}

	return nil
}
