package client

const (
	SplunkConfigType           = "splunkConfig"
	SplunkConfigFieldEnableTLS = "enableTLS"
	SplunkConfigFieldHost      = "host"
	SplunkConfigFieldPort      = "port"
	SplunkConfigFieldSource    = "source"
	SplunkConfigFieldToken     = "token"
)

type SplunkConfig struct {
	EnableTLS *bool  `json:"enableTLS,omitempty"`
	Host      string `json:"host,omitempty"`
	Port      *int64 `json:"port,omitempty"`
	Source    string `json:"source,omitempty"`
	Token     string `json:"token,omitempty"`
}
