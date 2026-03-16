package model

type ColumnTopValue struct {
	ID              int64
	ColumnStatID    int64
	Rank            int
	ValueJSON       []byte
	OccurrenceCount int64
}
