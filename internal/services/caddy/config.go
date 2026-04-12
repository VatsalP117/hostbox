package caddy

import "encoding/json"

// CaddyConfig is the top-level Caddy JSON configuration.
type CaddyConfig struct {
	Admin *CaddyAdmin `json:"admin,omitempty"`
	Apps  *CaddyApps  `json:"apps,omitempty"`
}

type CaddyAdmin struct {
	Listen string `json:"listen,omitempty"`
}

type CaddyApps struct {
	HTTP *CaddyHTTPApp `json:"http,omitempty"`
	TLS  *CaddyTLSApp  `json:"tls,omitempty"`
}

type CaddyHTTPApp struct {
	Servers map[string]*CaddyServer `json:"servers,omitempty"`
}

type CaddyServer struct {
	Listen         []string         `json:"listen,omitempty"`
	Routes         []CaddyRoute     `json:"routes,omitempty"`
	AutomaticHTTPS *CaddyAutoHTTPS  `json:"automatic_https,omitempty"`
	Logs           *CaddyServerLogs `json:"logs,omitempty"`
}

type CaddyAutoHTTPS struct {
	DisableRedirects bool     `json:"disable_redirects,omitempty"`
	Skip             []string `json:"skip,omitempty"`
	SkipCerts        []string `json:"skip_certificates,omitempty"`
}

type CaddyServerLogs struct {
	DefaultLoggerName string `json:"default_logger_name,omitempty"`
}

// CaddyRoute represents a single route rule.
type CaddyRoute struct {
	ID       string         `json:"@id,omitempty"`
	Group    string         `json:"group,omitempty"`
	Match    []CaddyMatch   `json:"match,omitempty"`
	Handle   []CaddyHandler `json:"handle,omitempty"`
	Terminal bool           `json:"terminal,omitempty"`
}

type CaddyMatch struct {
	Host []string `json:"host,omitempty"`
	Path []string `json:"path,omitempty"`
}

// CaddyHandler is a polymorphic handler. The "handler" field selects the Caddy module.
type CaddyHandler struct {
	Handler string `json:"handler"`

	// file_server
	Root       string   `json:"root,omitempty"`
	IndexNames []string `json:"index_names,omitempty"`

	// reverse_proxy
	Upstreams []CaddyUpstream `json:"upstreams,omitempty"`

	// encode
	Encodings *CaddyEncodings `json:"encodings,omitempty"`

	// rewrite
	URI string `json:"uri,omitempty"`

	// headers
	Response *CaddyHeaderOps `json:"response,omitempty"`

	// static_response
	StatusCode string `json:"status_code,omitempty"`
	Body       string `json:"body,omitempty"`

	// subroute
	Routes []CaddyRoute `json:"routes,omitempty"`
}

type CaddyUpstream struct {
	Dial string `json:"dial"`
}

type CaddyEncodings struct {
	Gzip *struct{} `json:"gzip,omitempty"`
	Zstd *struct{} `json:"zstd,omitempty"`
}

type CaddyHeaderOps struct {
	Set    map[string][]string `json:"set,omitempty"`
	Add    map[string][]string `json:"add,omitempty"`
	Delete []string            `json:"delete,omitempty"`
}

// --- TLS ---

type CaddyTLSApp struct {
	Automation *CaddyTLSAutomation `json:"automation,omitempty"`
}

type CaddyTLSAutomation struct {
	Policies []CaddyTLSPolicy `json:"policies,omitempty"`
}

type CaddyTLSPolicy struct {
	Subjects []string         `json:"subjects,omitempty"`
	Issuers  []CaddyTLSIssuer `json:"issuers,omitempty"`
}

type CaddyTLSIssuer struct {
	Module     string           `json:"module"`
	CA         string           `json:"ca,omitempty"`
	Email      string           `json:"email,omitempty"`
	Challenges *CaddyChallenges `json:"challenges,omitempty"`
}

type CaddyChallenges struct {
	DNS *CaddyDNSChallenge `json:"dns,omitempty"`
}

type CaddyDNSChallenge struct {
	Provider json.RawMessage `json:"provider,omitempty"`
}
