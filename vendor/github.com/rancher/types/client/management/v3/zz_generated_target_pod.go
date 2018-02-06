package client

const (
	TargetPodType                        = "targetPod"
	TargetPodFieldCondition              = "condition"
	TargetPodFieldRestartIntervalSeconds = "restartIntervalSeconds"
	TargetPodFieldRestartTimes           = "restartTimes"
)

type TargetPod struct {
	Condition              string `json:"condition,omitempty"`
	RestartIntervalSeconds *int64 `json:"restartIntervalSeconds,omitempty"`
	RestartTimes           *int64 `json:"restartTimes,omitempty"`
}
