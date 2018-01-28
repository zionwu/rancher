package client

const (
	TargetPodType              = "targetPod"
	TargetPodFieldIsRunning    = "isRunning"
	TargetPodFieldIsScheduled  = "isScheduled"
	TargetPodFieldRestartTimes = "restartTimes"
)

type TargetPod struct {
	IsRunning    *bool  `json:"isRunning,omitempty"`
	IsScheduled  *bool  `json:"isScheduled,omitempty"`
	RestartTimes *int64 `json:"restartTimes,omitempty"`
}
