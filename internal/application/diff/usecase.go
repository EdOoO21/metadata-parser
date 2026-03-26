package diff

import (
	"context"
	"fmt"
)

type DiffCatalogUseCase struct{}

func NewDiffCatalogUseCase() *DiffCatalogUseCase {
	return &DiffCatalogUseCase{}
}

func (uc *DiffCatalogUseCase) Execute(ctx context.Context, fromRunID, toRunID int64) (string, error) {
	_ = ctx

	return fmt.Sprintf(
		"diff generation is not implemented yet: from_run_id=%d to_run_id=%d",
		fromRunID,
		toRunID,
	), nil
}
