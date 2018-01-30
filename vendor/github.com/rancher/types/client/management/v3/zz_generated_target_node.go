package client

const (
	TargetNodeType               = "targetNode"
	TargetNodeFieldCPUThreshold  = "cpuThreshold"
	TargetNodeFieldCondition     = "condition"
	TargetNodeFieldDiskThreshold = "diskThreshold"
	TargetNodeFieldMemThreshold  = "memThreshold"
	TargetNodeFieldSelector      = "selector"
)

type TargetNode struct {
	CPUThreshold  *int64            `json:"cpuThreshold,omitempty"`
	Condition     string            `json:"condition,omitempty"`
	DiskThreshold *int64            `json:"diskThreshold,omitempty"`
	MemThreshold  *int64            `json:"memThreshold,omitempty"`
	Selector      map[string]string `json:"selector,omitempty"`
}
