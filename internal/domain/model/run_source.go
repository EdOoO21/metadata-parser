package model

import (
	"time"

	"github.com/EdOoO21/metadata-parser/internal/domain/types"
)

type RunSource struct {
	ID                  int64
	RunID               int64
	SourceID            int64
	StartedAt           time.Time
	FinishedAt          *time.Time
	Status              types.RunStatus
	ErrorMessage        *string
	EffectiveConfigJSON []byte
}
