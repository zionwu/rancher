package client

const (
	ClusterAlertSpecType                       = "clusterAlertSpec"
	ClusterAlertSpecFieldClusterId             = "clusterId"
	ClusterAlertSpecFieldDescription           = "description"
	ClusterAlertSpecFieldDisplayName           = "displayName"
	ClusterAlertSpecFieldInitialWaitSeconds    = "initialWaitSeconds"
	ClusterAlertSpecFieldRecipientList         = "recipientList"
	ClusterAlertSpecFieldRepeatIntervalSeconds = "repeatIntervalSeconds"
	ClusterAlertSpecFieldSeverity              = "severity"
	ClusterAlertSpecFieldTargetNode            = "targetNode"
	ClusterAlertSpecFieldTargetSystemService   = "targetSystemService"
)

type ClusterAlertSpec struct {
	ClusterId             string               `json:"clusterId,omitempty"`
	Description           string               `json:"description,omitempty"`
	DisplayName           string               `json:"displayName,omitempty"`
	InitialWaitSeconds    *int64               `json:"initialWaitSeconds,omitempty"`
	RecipientList         []Recipient          `json:"recipientList,omitempty"`
	RepeatIntervalSeconds *int64               `json:"repeatIntervalSeconds,omitempty"`
	Severity              string               `json:"severity,omitempty"`
	TargetNode            *TargetNode          `json:"targetNode,omitempty"`
	TargetSystemService   *TargetSystemService `json:"targetSystemService,omitempty"`
}
