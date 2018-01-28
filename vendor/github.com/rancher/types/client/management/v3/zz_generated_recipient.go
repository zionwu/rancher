package client

const (
	RecipientType                       = "recipient"
	RecipientFieldCustomPagerDutyConfig = "customPagerdutyConfig"
	RecipientFieldCustomWebhookConfig   = "customWebhookConfig"
	RecipientFieldNotifierId            = "notifierId"
	RecipientFieldRecipient             = "recipient"
)

type Recipient struct {
	CustomPagerDutyConfig *PagerdutyConfig `json:"customPagerdutyConfig,omitempty"`
	CustomWebhookConfig   *WebhookConfig   `json:"customWebhookConfig,omitempty"`
	NotifierId            string           `json:"notifierId,omitempty"`
	Recipient             string           `json:"recipient,omitempty"`
}
