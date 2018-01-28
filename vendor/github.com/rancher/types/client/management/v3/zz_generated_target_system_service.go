package client

const (
	TargetSystemServiceType      = "targetSystemService"
	TargetSystemServiceFieldType = "type"
)

type TargetSystemService struct {
	Type string `json:"type,omitempty"`
}
