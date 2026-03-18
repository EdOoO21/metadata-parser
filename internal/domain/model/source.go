package model

import (
	"time"

	"github.com/EdOoO21/metadata-parser/internal/domain/types"
)

type Source struct {
	ID          int64
	Name        string
	Kind        types.SourceKind
	Description *string
	CreatedAt   time.Time
}
