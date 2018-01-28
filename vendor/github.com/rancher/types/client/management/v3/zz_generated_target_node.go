package client

const (
	TargetNodeType               = "targetNode"
	TargetNodeFieldCPUThreshold  = "cpuThreshold"
	TargetNodeFieldDiskThreshold = "diskThreshold"
	TargetNodeFieldIsReady       = "isReady"
	TargetNodeFieldMemThreshold  = "memThreshold"
	TargetNodeFieldSelector      = "selector"
)

type TargetNode struct {
	CPUThreshold  *int64            `json:"cpuThreshold,omitempty"`
	DiskThreshold *int64            `json:"diskThreshold,omitempty"`
	IsReady       *bool             `json:"isReady,omitempty"`
	MemThreshold  *int64            `json:"memThreshold,omitempty"`
	Selector      map[string]string `json:"selector,omitempty"`
}
