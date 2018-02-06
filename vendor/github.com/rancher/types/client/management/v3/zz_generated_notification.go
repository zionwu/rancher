package client

const (
	NotificationType                 = "notification"
	NotificationFieldMessage         = "message"
	NotificationFieldPagerdutyConfig = "pagerdutyConfig"
	NotificationFieldSlackConfig     = "slackConfig"
	NotificationFieldSmtpConfig      = "smtpConfig"
	NotificationFieldWebhookConfig   = "webhookConfig"
)

type Notification struct {
	Message         string           `json:"message,omitempty"`
	PagerdutyConfig *PagerdutyConfig `json:"pagerdutyConfig,omitempty"`
	SlackConfig     *SlackConfig     `json:"slackConfig,omitempty"`
	SmtpConfig      *SmtpConfig      `json:"smtpConfig,omitempty"`
	WebhookConfig   *WebhookConfig   `json:"webhookConfig,omitempty"`
}
