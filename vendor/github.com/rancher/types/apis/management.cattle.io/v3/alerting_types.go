package v3

import (
	"github.com/rancher/norman/condition"
	"github.com/rancher/norman/types"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClusterAlert struct {
	types.Namespaced

	metav1.TypeMeta `json:",inline"`
	// Standard object’s metadata. More info:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ClusterAlertSpec `json:"spec"`
	// Most recent observed status of the cluster. More info:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#spec-and-status
	Status AlertStatus `json:"status"`
}

type ProjectAlert struct {
	types.Namespaced

	metav1.TypeMeta `json:",inline"`
	// Standard object’s metadata. More info:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec ProjectAlertSpec `json:"spec"`
	// Most recent observed status of the cluster. More info:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#spec-and-status
	Status AlertStatus `json:"status"`
}

type AlertCommonSpec struct {
	DisplayName           string      `json:"displayName,omitempty" norman:"required"`
	Description           string      `json:"description,omitempty"`
	Severity              string      `json:"severity,omitempty" norman:"required,options=info|critical|warning,default=critical"`
	RecipientList         []Recipient `json:"recipientList,omitempty" norman:"required"`
	InitialWaitSeconds    int         `json:"initialWaitSeconds,omitempty" norman:"required,default=180,min=0"`
	RepeatIntervalSeconds int         `json:"repeatIntervalSeconds,omitempty"  norman:"required,default=3600,min=0"`
}

type ClusterAlertSpec struct {
	AlertCommonSpec

	ClusterName         string              `json:"clusterName" norman:"type=reference[cluster]"`
	TargetNode          TargetNode          `json:"targetNode,omitempty"`
	TargetSystemService TargetSystemService `json:"targetSystemService,omitempty"`
}

type ProjectAlertSpec struct {
	AlertCommonSpec

	ProjectName    string         `json:"projectName" norman:"type=reference[project]"`
	TargetWorkload TargetWorkload `json:"targetWorkload,omitempty"`
	TargetPod      TargetPod      `json:"targetPod,omitempty"`
}

type Recipient struct {
	CustomPagerDutyConfig *PagerdutyConfig `json:"customPagerdutyConfig,omitempty"`
	CustomWebhookConfig   *WebhookConfig   `json:"customWebhookConfig,omitempty"`
	Recipient             string           `json:"recipient,omitempty"`
	NotifierId            string           `json:"notifierId,omitempty" norman:"type=reference[notifier]"`
}

type TargetNode struct {
	ID            string            `json:"id,omitempty"`
	Selector      map[string]string `json:"selector,omitempty"`
	IsReady       bool              `json:"isReady,omitempty"`
	DiskThreshold int               `json:"diskThreshold,omitempty" norman:"min=1,max=100"`
	MemThreshold  int               `json:"memThreshold,omitempty" norman:"min=1,max=100"`
	CPUThreshold  int               `json:"cpuThreshold,omitempty" norman:"min=1"`
}

type TargetPod struct {
	ID           string `json:"id,omitempty" norman:"required"`
	IsRunning    bool   `json:"isRunning,omitempty"`
	IsScheduled  bool   `json:"isScheduled,omitempty"`
	RestartTimes int    `json:"restartTimes,omitempty" norman:"min=1"`
}

type TargetWorkload struct {
	ID                    string            `json:"id,omitempty"`
	Selector              map[string]string `json:"selector,omitempty"`
	UnavailablePercentage int               `json:"unavailablePercentage,omitempty" norman:"required,min=1,max=100"`
}

type TargetSystemService struct {
	Type string `json:"type,omitempty" norman:"required,options=dns|etcd|controller manager|network|scheduler,default=scheduler"`
}

type AlertStatus struct {
	StartedAt string `json:"startedAt,omitempty"`
	State     string `json:"state,omitempty"`

	Conditions []AlertCondition `json:"conditions,omitempty"`
}

var (
	AlertConditionInitialized condition.Cond = "Initialized"
	AlertConditionProvisioned condition.Cond = "Provisioned"
)

type AlertCondition struct {
	// Type of cluster condition.
	Type condition.Cond `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status v1.ConditionStatus `json:"status"`
	// The last time this condition was updated.
	LastUpdateTime string `json:"lastUpdateTime,omitempty"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime string `json:"lastTransitionTime,omitempty"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty"`
	// Human-readable message indicating details about last transition
	Message string `json:"message,omitempty"`
}

type Notifier struct {
	types.Namespaced

	metav1.TypeMeta `json:",inline"`
	// Standard object’s metadata. More info:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec NotifierSpec `json:"spec"`
	// Most recent observed status of the cluster. More info:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#spec-and-status
	Status NotifierStatus `json:"status"`
}

type NotifierSpec struct {
	ClusterName string `json:"clusterName" norman:"type=reference[cluster]"`

	DisplayName     string           `json:"displayName,omitempty" norman:"required"`
	Description     string           `json:"description,omitempty"`
	SmtpConfig      *SmtpConfig      `json:"smtpConfig,omitempty"`
	SlackConfig     *SlackConfig     `json:"slackConfig,omitempty"`
	PagerdutyConfig *PagerdutyConfig `json:"pagerdutyConfig,omitempty"`
	WebhookConfig   *WebhookConfig   `json:"webhookConfig,omitempty"`
}

type SmtpConfig struct {
	Host             string `json:"host,omitempty" norman:"required,type=dnsLabel"`
	Port             int    `json:"port,omitempty" norman:"required,min=1,max=65535"`
	Username         string `json:"username,omitempty" norman:"required"`
	Password         string `json:"password,omitempty" norman:"required,type=masked"`
	DefaultRecipient string `json:"defaultRecipient,omitempty" norman:"required"`
	TLS              bool   `json:"tls,omitempty" norman:"required,default=true"`
}

type SlackConfig struct {
	URL string `json:"url,omitempty" norman:"required"`
}

type PagerdutyConfig struct {
	ServiceKey string `json:"serviceKey,omitempty" norman:"required"`
}

type WebhookConfig struct {
	URL string `json:"url,omitempty" norman:"required"`
}

type NotifierStatus struct {
}
