package client

const (
	TargetPodType              = "targetPod"
	TargetPodFieldCondition    = "condition"
	TargetPodFieldRestartTimes = "restartTimes"
)

type TargetPod struct {
	Condition    string `json:"condition,omitempty"`
	RestartTimes *int64 `json:"restartTimes,omitempty"`
}
