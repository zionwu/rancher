package client

const (
	SlackConfigType         = "slackConfig"
	SlackConfigFieldChannel = "channel"
	SlackConfigFieldURL     = "url"
)

type SlackConfig struct {
	Channel string `json:"channel,omitempty"`
	URL     string `json:"url,omitempty"`
}
