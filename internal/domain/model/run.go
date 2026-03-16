package model

import (
	"time"

	"catalog-tool/internal/domain/types"
)

type Run struct {
	ID                 int64
	StartedAt          time.Time
	FinishedAt         *time.Time
	Status             types.RunStatus
	ConfigHash         string
	ConfigSnapshotJSON []byte
	ErrorMessage       *string
}
