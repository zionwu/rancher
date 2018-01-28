package client

const (
	SlackConfigType     = "slackConfig"
	SlackConfigFieldURL = "url"
)

type SlackConfig struct {
	URL string `json:"url,omitempty"`
}
