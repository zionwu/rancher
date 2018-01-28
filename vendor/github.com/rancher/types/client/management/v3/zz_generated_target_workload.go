package client

const (
	TargetWorkloadType                       = "targetWorkload"
	TargetWorkloadFieldSelector              = "selector"
	TargetWorkloadFieldUnavailablePercentage = "unavailablePercentage"
)

type TargetWorkload struct {
	Selector              map[string]string `json:"selector,omitempty"`
	UnavailablePercentage *int64            `json:"unavailablePercentage,omitempty"`
}
