package ports

import (
	"context"

	"github.com/EdOoO21/metadata-parser/internal/application/contracts"
	"github.com/EdOoO21/metadata-parser/internal/settings"
)

type CSVParser interface {
	// ParseSource читает файловый CSV-источник и возвращает найденные датасеты в общем контракте сканирования.
	ParseSource(ctx context.Context, src settings.SourceConfig) (*contracts.SourceScanResult, error)
}
