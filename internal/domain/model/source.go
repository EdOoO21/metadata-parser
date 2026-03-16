package model

import (
	"time"

	"catalog-tool/internal/domain/types"
)

type Source struct {
	ID          int64
	Name        string
	Kind        types.SourceKind
	Description *string
	CreatedAt   time.Time
}
