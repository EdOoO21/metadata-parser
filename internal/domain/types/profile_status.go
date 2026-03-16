package types

type ProfileStatus string

const (
	ProfileStatusProfiled             ProfileStatus = "profiled"
	ProfileStatusDiscoveredOnly       ProfileStatus = "discovered_only"
	ProfileStatusSkippedRequiresParams ProfileStatus = "skipped_requires_params"
	ProfileStatusFailed               ProfileStatus = "failed"
)
