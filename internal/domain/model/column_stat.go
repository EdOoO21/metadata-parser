package model

type ColumnStat struct {
	ID            int64
	ColumnID      int64
	NonNullCount  int64
	NullCount     int64
	DistinctCount int64
	MinValueJSON  []byte
	MaxValueJSON  []byte
}
