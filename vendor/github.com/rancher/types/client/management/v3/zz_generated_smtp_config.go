package client

const (
	SmtpConfigType                  = "smtpConfig"
	SmtpConfigFieldDefaultRecipient = "defaultRecipient"
	SmtpConfigFieldHost             = "host"
	SmtpConfigFieldPassword         = "password"
	SmtpConfigFieldPort             = "port"
	SmtpConfigFieldTLS              = "tls"
	SmtpConfigFieldUsername         = "username"
)

type SmtpConfig struct {
	DefaultRecipient string `json:"defaultRecipient,omitempty"`
	Host             string `json:"host,omitempty"`
	Password         string `json:"password,omitempty"`
	Port             *int64 `json:"port,omitempty"`
	TLS              *bool  `json:"tls,omitempty"`
	Username         string `json:"username,omitempty"`
}
