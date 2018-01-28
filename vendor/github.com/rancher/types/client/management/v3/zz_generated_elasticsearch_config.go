package client

const (
	ElasticsearchConfigType              = "elasticsearchConfig"
	ElasticsearchConfigFieldAuthPassword = "authPassword"
	ElasticsearchConfigFieldAuthUserName = "authUsername"
	ElasticsearchConfigFieldDateFormat   = "dateFormat"
<<<<<<< HEAD
	ElasticsearchConfigFieldEndpoint     = "endpoint"
=======
	ElasticsearchConfigFieldEnableTLS    = "enableTLS"
	ElasticsearchConfigFieldHost         = "host"
>>>>>>> update types
	ElasticsearchConfigFieldIndexPrefix  = "indexPrefix"
)

type ElasticsearchConfig struct {
	AuthPassword string `json:"authPassword,omitempty"`
	AuthUserName string `json:"authUsername,omitempty"`
	DateFormat   string `json:"dateFormat,omitempty"`
<<<<<<< HEAD
	Endpoint     string `json:"endpoint,omitempty"`
=======
	EnableTLS    *bool  `json:"enableTLS,omitempty"`
	Host         string `json:"host,omitempty"`
>>>>>>> update types
	IndexPrefix  string `json:"indexPrefix,omitempty"`
}
