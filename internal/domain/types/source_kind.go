package types

type SourceKind string

const (
	SourceKindPostgres SourceKind = "postgres"
	SourceKindFiles    SourceKind = "files"
	SourceKindREST     SourceKind = "rest"
)
