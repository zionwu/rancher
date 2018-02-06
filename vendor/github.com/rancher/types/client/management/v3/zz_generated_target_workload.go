package client

const (
	TargetWorkloadType                       = "targetWorkload"
	TargetWorkloadFieldSelector              = "selector"
	TargetWorkloadFieldType                  = "type"
	TargetWorkloadFieldUnavailablePercentage = "unavailablePercentage"
)

type TargetWorkload struct {
	Selector              map[string]string `json:"selector,omitempty"`
	Type                  string            `json:"type,omitempty"`
	UnavailablePercentage *int64            `json:"unavailablePercentage,omitempty"`
}
