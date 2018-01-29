package client

const (
<<<<<<< HEAD
	SlackConfigType                  = "slackConfig"
	SlackConfigFieldDefaultRecipient = "defaultRecipient"
	SlackConfigFieldURL              = "url"
)

type SlackConfig struct {
	DefaultRecipient string `json:"defaultRecipient,omitempty"`
	URL              string `json:"url,omitempty"`
=======
	SlackConfigType         = "slackConfig"
	SlackConfigFieldChannel = "channel"
	SlackConfigFieldURL     = "url"
)

type SlackConfig struct {
	Channel string `json:"channel,omitempty"`
	URL     string `json:"url,omitempty"`
>>>>>>> update types for alerting
}
