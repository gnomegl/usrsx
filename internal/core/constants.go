package core

const (
	WMNRemoteURL = "https://raw.githubusercontent.com/WebBreacher/WhatsMyName/main/wmn-data.json"
	WMNSchemaURL = "https://raw.githubusercontent.com/WebBreacher/WhatsMyName/main/wmn-data-schema.json"

	HTTPRequestTimeoutSeconds = 30
	HTTPSSLVerify             = false
	HTTPAllowRedirects        = false

	MaxConcurrentTasks = 50

	MinTasks      = 1
	MaxTasksLimit = 1000
	MinTimeout    = 0
	MaxTimeout    = 300

	HighConcurrencyThreshold      = 100
	HighConcurrencyMinTimeout     = 10
	VeryHighConcurrencyThreshold  = 50
	VeryHighConcurrencyMinTimeout = 5
	ExtremeConcurrencyThreshold   = 500
	LowTimeoutWarningThreshold    = 3

	AccountPlaceholder = "{account}"

	Version     = "2.0.0"
	Description = "The most powerful and fast username availability checker (Go version)"
)
