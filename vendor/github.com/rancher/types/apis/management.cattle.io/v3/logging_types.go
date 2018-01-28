package v3

import (
	"github.com/rancher/norman/condition"
	"github.com/rancher/norman/types"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ClusterLogging struct {
	types.Namespaced

	metav1.TypeMeta `json:",inline"`
	// Standard object’s metadata. More info:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Specification of the desired behavior of the the cluster. More info:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#spec-and-status
	Spec ClusterLoggingSpec `json:"spec"`
	// Most recent observed status of the cluster. More info:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#spec-and-status
	Status LoggingStatus `json:"status"`
}

type ProjectLogging struct {
	types.Namespaced

	metav1.TypeMeta `json:",inline"`
	// Standard object’s metadata. More info:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#metadata
	metav1.ObjectMeta `json:"metadata,omitempty"`
	// Specification of the desired behavior of the the cluster. More info:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#spec-and-status
	Spec ProjectLoggingSpec `json:"spec"`
	// Most recent observed status of the cluster. More info:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/api-conventions.md#spec-and-status
	Status LoggingStatus `json:"status"`
}

type LoggingCommonSpec struct {
	DisplayName string `json:"displayName,omitempty"`

	OutputFlushInterval int               `json:"outputFlushInterval" norman:"default=3"`
	OutputTags          map[string]string `json:"outputTags"`

	ElasticsearchConfig *ElasticsearchConfig `json:"elasticsearchConfig,omitempty"`
	SplunkConfig        *SplunkConfig        `json:"splunkConfig,omitempty"`
	KafkaConfig         *KafkaConfig         `json:"kafkaConfig,omitempty"`
	SyslogConfig        *SyslogConfig        `json:"syslogConfig,omitempty"`
}

type ClusterLoggingSpec struct {
	LoggingCommonSpec
	ClusterName string `json:"clusterName" norman:"type=reference[cluster]"`

	EmbeddedConfig *EmbeddedConfig `json:"embeddedConfig,omitempty"`
}

type ProjectLoggingSpec struct {
	LoggingCommonSpec

	ProjectName string `json:"projectName" norman:"type=reference[project]"`
}

type LoggingStatus struct {
	Conditions []LoggingCondition `json:"conditions,omitempty"`
}

var (
	ClusterLoggingConditionInitialized condition.Cond = "Initialized"
	ClusterLoggingConditionProvisioned condition.Cond = "Provisioned"
)

type LoggingCondition struct {
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

type ElasticsearchConfig struct {
	Host         string `json:"host,omitempty" norman:"required"`
	Port         int    `json:"port,omitempty" norman:"required,default=9200,min=1,max=65535"`
	EnableTLS    bool   `json:"enableTLS,omitempty"`
	IndexPrefix  string `json:"indexPrefix,omitempty" norman:"required"`
	DateFormat   string `json:"dateFormat,omitempty" norman:"required,type=enum,options=YYYY-MM-DD|YYYY-MM|YYYY,default=YYYY-MM-DD"`
	AuthUserName string `json:"authUsername,omitempty"` //secret
	AuthPassword string `json:"authPassword,omitempty"` //secret
}

type SplunkConfig struct {
	Host      string `json:"host,omitempty" norman:"required"`
	Port      int    `json:"port,omitempty" norman:"required,default=8088,min=1,max=65535"`
	EnableTLS bool   `json:"enableTLS,omitempty" norman:"default=false"`
	Source    string `json:"source,omitempty"`
	Token     string `json:"token,omitempty" norman:"required"` //secret
}

type EmbeddedConfig struct {
	IndexPrefix string `json:"indexPrefix,omitempty" norman:"required"`
	DateFormat  string `json:"dateFormat,omitempty" norman:"required,type=enum,options=YYYY-MM-DD|YYYY-MM|YYYY,default=YYYY-MM-DD"`
}

type KafkaConfig struct {
	Zookeeper      *Zookeeper  `json:"zookeeper,omitempty"`
	Broker         *BrokerList `json:"broker,omitempty"`
	Topic          string      `json:"topic,omitempty" norman:"required"`
	DataType       string      `json:"dataType,omitempty" norman:"required,type=enum,options=json|ltsv|msgpack,default=json"`
	MaxSendRetries int         `json:"maxSendRetries,omitempty" norman:"default=3"`
}

type Zookeeper struct {
	Host string `json:"host,omitempty"`
	Port int    `json:"port,omitempty" norman:"default=2181,min=1,max=65535"`
}

type BrokerList struct {
	BrokerList []string `json:"brokerList,omitempty"`
}

type SyslogConfig struct {
	Host     string `json:"host,omitempty" norman:"required"`
	Port     int    `json:"port,omitempty" norman:"required,default=51400,min=1,max=65535"`
	Severity string `json:"severity,omitempty" norman:"default=notice,type=enum,options=emerg|alert|crit|err|warning|notice|info|debug"`
	Program  string `json:"program,omitempty"`
}
