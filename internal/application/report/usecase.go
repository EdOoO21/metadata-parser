package report

import (
	"context"
	"fmt"
)

type ReportCatalogUseCase struct{}

func NewReportCatalogUseCase() *ReportCatalogUseCase {
	return &ReportCatalogUseCase{}
}

func (uc *ReportCatalogUseCase) Execute(ctx context.Context, runID int64) (string, error) {
	_ = ctx

	return fmt.Sprintf("report generation is not implemented yet: run_id=%d", runID), nil
}
