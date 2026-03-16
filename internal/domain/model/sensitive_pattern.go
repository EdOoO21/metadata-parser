package model

import "time"

type SensitivePattern struct {
	ID          int64
	Name        string
	Pattern     string
	Description *string
	IsActive    bool
	CreatedAt   time.Time
}
