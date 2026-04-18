package caddy

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/VatsalP117/hostbox/internal/platform/hostnames"
)

// BuilderConfig holds platform-level settings for config generation.
type BuilderConfig struct {
	PlatformDomain  string
	DashboardDomain string
	PlatformHTTPS   bool
	ACMEEmail       string
	APIUpstream     string // "localhost:8080"
	DeploymentRoot  string // "/app/deployments"
	DNSProvider     string
	DNSProviderConf json.RawMessage
}

// ConfigBuilder assembles a CaddyConfig from deployments and domains.
type ConfigBuilder struct {
	cfg BuilderConfig
}

func NewConfigBuilder(cfg BuilderConfig) *ConfigBuilder {
	return &ConfigBuilder{cfg: cfg}
}

// BuildFullConfig constructs the complete Caddy JSON config from current state.
func (b *ConfigBuilder) BuildFullConfig(
	deployments []ActiveDeployment,
	domains []VerifiedDomain,
) *CaddyConfig {
	routes := make([]CaddyRoute, 0, len(deployments)+len(domains)+1)

	// 1. Platform route (highest priority)
	routes = append(routes, b.buildPlatformRoute())

	// 2. Custom domain routes
	for _, d := range domains {
		routes = append(routes, b.buildCustomDomainRoute(d))
	}

	// 3. Project routes: production + branch + preview
	projectDeployments := groupByProject(deployments)
	for _, projDeploys := range projectDeployments {
		routes = append(routes, b.buildProjectRoutes(projDeploys)...)
	}

	config := &CaddyConfig{
		Admin: &CaddyAdmin{Listen: ":2019"},
		Apps: &CaddyApps{
			HTTP: &CaddyHTTPApp{
				Servers: map[string]*CaddyServer{
					"main": {
						Listen: []string{":443", ":80"},
						Routes: routes,
					},
				},
			},
			TLS: b.buildTLSConfig(domains),
		},
	}

	return config
}

func (b *ConfigBuilder) buildPlatformRoute() CaddyRoute {
	hosts := []string{b.cfg.DashboardDomain}
	if b.cfg.PlatformDomain != b.cfg.DashboardDomain {
		hosts = append(hosts, b.cfg.PlatformDomain)
	}

	return CaddyRoute{
		ID:    "route-platform",
		Group: "platform",
		Match: []CaddyMatch{{Host: hosts}},
		Handle: []CaddyHandler{
			{
				Handler: "subroute",
				Routes: []CaddyRoute{
					{
						Handle: []CaddyHandler{
							{Handler: "encode", Encodings: &CaddyEncodings{Gzip: &struct{}{}}},
							{
								Handler:   "reverse_proxy",
								Upstreams: []CaddyUpstream{{Dial: b.cfg.APIUpstream}},
							},
						},
					},
				},
			},
		},
		Terminal: true,
	}
}

func (b *ConfigBuilder) buildPreviewRoute(d ActiveDeployment) CaddyRoute {
	host := hostnames.PreviewHost(d.ProjectSlug, d.DeploymentID, b.cfg.PlatformDomain)

	return CaddyRoute{
		ID:       fmt.Sprintf("route-deploy-%s", d.DeploymentID),
		Match:    []CaddyMatch{{Host: []string{host}}},
		Handle:   b.buildFileServerHandlers(d.ArtifactPath, d.Framework),
		Terminal: true,
	}
}

func (b *ConfigBuilder) buildProductionRoute(projectSlug, projectID, artifactPath, framework string) CaddyRoute {
	host := hostnames.ProductionHost(projectSlug, b.cfg.PlatformDomain)

	return CaddyRoute{
		ID:       fmt.Sprintf("route-prod-%s", projectID),
		Match:    []CaddyMatch{{Host: []string{host}}},
		Handle:   b.buildFileServerHandlers(artifactPath, framework),
		Terminal: true,
	}
}

func (b *ConfigBuilder) buildBranchStableRoute(projectSlug, projectID, branchSlug, artifactPath, framework string) CaddyRoute {
	host := hostnames.BranchHost(projectSlug, branchSlug, b.cfg.PlatformDomain)

	return CaddyRoute{
		ID:       fmt.Sprintf("route-branch-%s-%s", projectID, branchSlug),
		Match:    []CaddyMatch{{Host: []string{host}}},
		Handle:   b.buildFileServerHandlers(artifactPath, framework),
		Terminal: true,
	}
}

func (b *ConfigBuilder) buildCustomDomainRoute(d VerifiedDomain) CaddyRoute {
	hosts := []string{d.Domain}
	if !strings.HasPrefix(d.Domain, "www.") {
		hosts = append(hosts, "www."+d.Domain)
	}

	return CaddyRoute{
		ID:       fmt.Sprintf("route-domain-%s", d.DomainID),
		Match:    []CaddyMatch{{Host: hosts}},
		Handle:   b.buildFileServerHandlers(d.ProductionArtifact, d.Framework),
		Terminal: true,
	}
}

func (b *ConfigBuilder) buildProjectRoutes(deploys []ActiveDeployment) []CaddyRoute {
	var routes []CaddyRoute
	branchLatest := make(map[string]ActiveDeployment)

	for _, d := range deploys {
		if d.IsProduction {
			routes = append(routes, b.buildProductionRoute(d.ProjectSlug, d.ProjectID, d.ArtifactPath, d.Framework))
		}

		// Track latest deployment per branch for branch-stable routes
		slug := Slugify(d.Branch)
		if existing, ok := branchLatest[slug]; !ok || d.DeploymentID > existing.DeploymentID {
			branchLatest[slug] = d
		}

		// Every deployment gets a preview route
		routes = append(routes, b.buildPreviewRoute(d))
	}

	// Branch-stable routes
	for slug, d := range branchLatest {
		routes = append(routes, b.buildBranchStableRoute(d.ProjectSlug, d.ProjectID, slug, d.ArtifactPath, d.Framework))
	}

	return routes
}

func (b *ConfigBuilder) buildFileServerHandlers(artifactPath, framework string) []CaddyHandler {
	handlers := []CaddyHandler{
		{
			Handler:   "encode",
			Encodings: &CaddyEncodings{Gzip: &struct{}{}, Zstd: &struct{}{}},
		},
		{
			Handler: "headers",
			Response: &CaddyHeaderOps{
				Set: map[string][]string{
					"X-Frame-Options":        {"DENY"},
					"X-Content-Type-Options": {"nosniff"},
					"Referrer-Policy":        {"strict-origin-when-cross-origin"},
				},
			},
		},
	}

	if isSPAFramework(framework) {
		handlers = append(handlers, CaddyHandler{
			Handler: "subroute",
			Routes: []CaddyRoute{
				{
					Match: []CaddyMatch{{Path: []string{
						"/_next/static/*",
						"/static/*",
						"/assets/*",
						"*.js",
						"*.css",
						"*.woff2",
						"*.woff",
					}}},
					Handle: []CaddyHandler{
						{
							Handler: "headers",
							Response: &CaddyHeaderOps{
								Set: map[string][]string{
									"Cache-Control": {"public, max-age=31536000, immutable"},
								},
							},
						},
						{Handler: "file_server", Root: artifactPath},
					},
					Terminal: true,
				},
				{
					Handle: []CaddyHandler{
						{Handler: "rewrite", URI: "{http.request.uri.path}"},
						{Handler: "file_server", Root: artifactPath},
					},
				},
			},
		})
	} else {
		handlers = append(handlers, CaddyHandler{
			Handler: "file_server",
			Root:    artifactPath,
		})
	}

	return handlers
}

func (b *ConfigBuilder) buildTLSConfig(domains []VerifiedDomain) *CaddyTLSApp {
	policies := []CaddyTLSPolicy{
		{
			Subjects: []string{b.cfg.DashboardDomain, b.cfg.PlatformDomain},
			Issuers: []CaddyTLSIssuer{{
				Module: "acme",
				Email:  b.cfg.ACMEEmail,
			}},
		},
	}

	if b.cfg.DNSProvider != "" && len(b.cfg.DNSProviderConf) > 0 {
		policies = append(policies, CaddyTLSPolicy{
			Subjects: []string{"*." + b.cfg.PlatformDomain},
			Issuers: []CaddyTLSIssuer{{
				Module: "acme",
				Email:  b.cfg.ACMEEmail,
				Challenges: &CaddyChallenges{
					DNS: &CaddyDNSChallenge{
						Provider: b.cfg.DNSProviderConf,
					},
				},
			}},
		})
	}

	return &CaddyTLSApp{
		Automation: &CaddyTLSAutomation{
			Policies: policies,
		},
	}
}
