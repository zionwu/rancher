package client

const (
	NotifierSpecType                 = "notifierSpec"
	NotifierSpecFieldClusterId       = "clusterId"
	NotifierSpecFieldDescription     = "description"
	NotifierSpecFieldDisplayName     = "displayName"
	NotifierSpecFieldPagerdutyConfig = "pagerdutyConfig"
	NotifierSpecFieldSlackConfig     = "slackConfig"
	NotifierSpecFieldSmtpConfig      = "smtpConfig"
	NotifierSpecFieldWebhookConfig   = "webhookConfig"
)

type NotifierSpec struct {
	ClusterId       string           `json:"clusterId,omitempty"`
	Description     string           `json:"description,omitempty"`
	DisplayName     string           `json:"displayName,omitempty"`
	PagerdutyConfig *PagerdutyConfig `json:"pagerdutyConfig,omitempty"`
	SlackConfig     *SlackConfig     `json:"slackConfig,omitempty"`
	SmtpConfig      *SmtpConfig      `json:"smtpConfig,omitempty"`
	WebhookConfig   *WebhookConfig   `json:"webhookConfig,omitempty"`
}
