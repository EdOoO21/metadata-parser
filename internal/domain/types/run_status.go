package types

type RunStatus string

const (
	RunStatusRunning RunStatus = "running"
	RunStatusSuccess RunStatus = "success"
	RunStatusPartial RunStatus = "partial"
	RunStatusFailed  RunStatus = "failed"
)
