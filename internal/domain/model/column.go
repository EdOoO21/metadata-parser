package model

import "github.com/EdOoO21/metadata-parser/internal/domain/types"

type Column struct {
	ID              int64
	DatasetID       int64
	Name            string
	OriginalType    string
	NormalizedType  types.CanonicalType
	IsNullable      bool
	Comment         *string
	OrdinalPosition int
}
