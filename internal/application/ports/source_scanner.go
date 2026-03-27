package ports

import (
	"context"

	"github.com/EdOoO21/metadata-parser/internal/application/contracts"
	"github.com/EdOoO21/metadata-parser/internal/settings"
)

// SourceScanner читает один source и возвращает найденные датасеты в общем контракте сканирования.
type SourceScanner interface {
	ParseSource(ctx context.Context, src settings.SourceConfig) (*contracts.SourceScanResult, error)
}

// SourceScannerFactory подбирает нужный scanner для конкретного source.
type SourceScannerFactory interface {
	ForSource(src settings.SourceConfig) (SourceScanner, error)
}
