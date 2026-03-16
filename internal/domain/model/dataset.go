package model

import (
	"time"

	"catalog-tool/internal/domain/types"
)

type Dataset struct {
	ID            int64
	RunSourceID   int64
	Kind          types.DatasetKind
	DatasetKey    string
	Name          string
	Location      string
	Comment       *string
	RowCount      *int64
	DiscoveredAt  time.Time
	ProfileStatus types.ProfileStatus
	ProfileError  *string
	MetadataJSON  []byte
}
