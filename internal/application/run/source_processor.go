package run

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	appports "github.com/EdOoO21/metadata-parser/internal/application/ports"
	"github.com/EdOoO21/metadata-parser/internal/domain/model"
	"github.com/EdOoO21/metadata-parser/internal/domain/types"
	"github.com/EdOoO21/metadata-parser/internal/ports"
	"github.com/EdOoO21/metadata-parser/internal/settings"
)

type SourceProcessor struct {
	logger             ports.Logger
	filesSourceHandler *FilesSourceHandler
}

func NewSourceProcessor(logger ports.Logger, filesSourceHandler *FilesSourceHandler) *SourceProcessor {
	return &SourceProcessor{
		logger:             logger,
		filesSourceHandler: filesSourceHandler,
	}
}

func (p *SourceProcessor) Process(
	ctx context.Context,
	repo appports.CatalogRepository,
	runID int64,
	src settings.SourceConfig,
) error {
	if p.logger != nil {
		p.logger.Info("configured source discovered",
			"name", src.Name,
			"kind", src.Kind,
		)
	}

	source, err := repo.EnsureSource(ctx, model.Source{
		Name: src.Name,
		Kind: types.SourceKind(src.Kind),
	})
	if err != nil {
		return fmt.Errorf("ensure source: %w", err)
	}

	effectiveConfigJSON, err := json.Marshal(src.Config)
	if err != nil {
		return fmt.Errorf("marshal effective config: %w", err)
	}

	runSource, err := repo.CreateRunSource(ctx, model.RunSource{
		RunID:               runID,
		SourceID:            source.ID,
		StartedAt:           time.Now().UTC(),
		Status:              types.RunStatusRunning,
		EffectiveConfigJSON: effectiveConfigJSON,
	})
	if err != nil {
		return fmt.Errorf("create run source: %w", err)
	}

	processErr := p.handleByKind(ctx, repo, runSource.ID, src)

	finishedAt := time.Now().UTC()

	if processErr != nil {
		msg := processErr.Error()
		if err := repo.UpdateRunSourceStatus(ctx, runSource.ID, types.RunStatusFailed, &finishedAt, &msg); err != nil {
			return fmt.Errorf("processing error: %v; update run source status: %w", processErr, err)
		}
		return processErr
	}

	if err := repo.UpdateRunSourceStatus(ctx, runSource.ID, types.RunStatusSuccess, &finishedAt, nil); err != nil {
		return fmt.Errorf("update run source status: %w", err)
	}

	if p.logger != nil {
		p.logger.Info("source processed successfully",
			"name", src.Name,
			"kind", src.Kind,
			"run_source_id", runSource.ID,
		)
	}

	return nil
}

func (p *SourceProcessor) handleByKind(
	ctx context.Context,
	repo appports.CatalogRepository,
	runSourceID int64,
	src settings.SourceConfig,
) error {
	switch src.Kind {
	case string(types.SourceKindFiles):
		if p.filesSourceHandler == nil {
			return fmt.Errorf("files source handler is not configured")
		}
		return p.filesSourceHandler.Handle(ctx, repo, runSourceID, src)
	default:
		return fmt.Errorf("source kind %q is not implemented yet", src.Kind)
	}
}
