package client

const (
	AlertStatusType            = "alertStatus"
	AlertStatusFieldConditions = "conditions"
	AlertStatusFieldStartedAt  = "startedAt"
	AlertStatusFieldState      = "state"
)

type AlertStatus struct {
	Conditions []AlertCondition `json:"conditions,omitempty"`
	StartedAt  string           `json:"startedAt,omitempty"`
	State      string           `json:"state,omitempty"`
}
