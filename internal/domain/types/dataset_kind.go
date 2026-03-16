package types

type DatasetKind string

const (
	DatasetKindTable    DatasetKind = "table"
	DatasetKindView     DatasetKind = "view"
	DatasetKindFile     DatasetKind = "file"
	DatasetKindEndpoint DatasetKind = "endpoint"
)
