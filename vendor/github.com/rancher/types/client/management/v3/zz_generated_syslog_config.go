package client

const (
	SyslogConfigType          = "syslogConfig"
	SyslogConfigFieldEndpoint = "endpoint"
	SyslogConfigFieldProgram  = "program"
	SyslogConfigFieldSeverity = "severity"
)

type SyslogConfig struct {
	Endpoint string `json:"endpoint,omitempty"`
	Program  string `json:"program,omitempty"`
	Severity string `json:"severity,omitempty"`
}
