package types

type CanonicalType string

const (
	CanonicalTypeString    CanonicalType = "STRING"
	CanonicalTypeNumber    CanonicalType = "NUMBER"
	CanonicalTypeBoolean   CanonicalType = "BOOLEAN"
	CanonicalTypeTimestamp CanonicalType = "TIMESTAMP"
	CanonicalTypeArray     CanonicalType = "ARRAY"
)
