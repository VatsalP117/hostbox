package caddy

import (
	"encoding/json"
	"testing"
)

func newTestBuilder() *ConfigBuilder {
	return NewConfigBuilder(BuilderConfig{
		PlatformDomain: "hostbox.example.com",
		PlatformHTTPS:  true,
		ACMEEmail:      "admin@example.com",
		APIUpstream:    "localhost:8080",
		DeploymentRoot: "/app/deployments",
	})
}

func TestBuildFullConfig_PlatformRoute(t *testing.T) {
	b := newTestBuilder()
	config := b.BuildFullConfig(nil, nil)

	if config.Admin.Listen != "localhost:2019" {
		t.Errorf("admin listen = %q, want localhost:2019", config.Admin.Listen)
	}

	servers := config.Apps.HTTP.Servers
	if len(servers) != 1 {
		t.Fatalf("expected 1 server, got %d", len(servers))
	}

	main := servers["main"]
	if main == nil {
		t.Fatal("missing 'main' server")
	}
	if len(main.Routes) < 1 {
		t.Fatal("expected at least 1 route (platform)")
	}

	platform := main.Routes[0]
	if platform.ID != "route-platform" {
		t.Errorf("platform route id = %q", platform.ID)
	}
	if platform.Match[0].Host[0] != "hostbox.example.com" {
		t.Errorf("platform host = %q", platform.Match[0].Host[0])
	}
	if !platform.Terminal {
		t.Error("platform route should be terminal")
	}
}

func TestBuildFullConfig_DeploymentRoutes(t *testing.T) {
	b := newTestBuilder()

	deploys := []ActiveDeployment{
		{
			DeploymentID: "dpl_abc12345",
			ProjectID:    "prj_001",
			ProjectSlug:  "my-app",
			Branch:       "main",
			CommitSHA:    "abc123",
			IsProduction: true,
			ArtifactPath: "/app/deployments/prj_001/dpl_abc12345",
			Framework:    "vite",
		},
		{
			DeploymentID: "dpl_def67890",
			ProjectID:    "prj_001",
			ProjectSlug:  "my-app",
			Branch:       "feature/login",
			CommitSHA:    "def456",
			IsProduction: false,
			ArtifactPath: "/app/deployments/prj_001/dpl_def67890",
			Framework:    "vite",
		},
	}

	config := b.BuildFullConfig(deploys, nil)
	routes := config.Apps.HTTP.Servers["main"].Routes

	// Should have: platform + production + 2 preview + 2 branch-stable
	if len(routes) < 4 {
		t.Fatalf("expected at least 4 routes, got %d", len(routes))
	}

	// Find the production route
	var foundProd, foundPreview1, foundPreview2 bool
	for _, r := range routes {
		switch r.ID {
		case "route-prod-prj_001":
			foundProd = true
			if r.Match[0].Host[0] != "my-app.hostbox.example.com" {
				t.Errorf("production host = %q", r.Match[0].Host[0])
			}
		case "route-deploy-dpl_abc12345":
			foundPreview1 = true
			if r.Match[0].Host[0] != "my-app-dpl_abc1.hostbox.example.com" {
				t.Errorf("preview1 host = %q", r.Match[0].Host[0])
			}
		case "route-deploy-dpl_def67890":
			foundPreview2 = true
			if r.Match[0].Host[0] != "my-app-dpl_def6.hostbox.example.com" {
				t.Errorf("preview2 host = %q", r.Match[0].Host[0])
			}
		}
	}

	if !foundProd {
		t.Error("missing production route")
	}
	if !foundPreview1 {
		t.Error("missing preview route for dpl_abc12345")
	}
	if !foundPreview2 {
		t.Error("missing preview route for dpl_def67890")
	}
}

func TestBuildFullConfig_CustomDomainRoutes(t *testing.T) {
	b := newTestBuilder()

	domains := []VerifiedDomain{
		{
			DomainID:           "dom_001",
			Domain:             "myapp.com",
			ProjectID:          "prj_001",
			ProjectSlug:        "my-app",
			ProductionArtifact: "/app/deployments/prj_001/dpl_prod",
			Framework:          "hugo",
		},
	}

	config := b.BuildFullConfig(nil, domains)
	routes := config.Apps.HTTP.Servers["main"].Routes

	// Should have platform + custom domain
	if len(routes) < 2 {
		t.Fatalf("expected at least 2 routes, got %d", len(routes))
	}

	domRoute := routes[1] // custom domain is after platform
	if domRoute.ID != "route-domain-dom_001" {
		t.Errorf("domain route id = %q", domRoute.ID)
	}

	hosts := domRoute.Match[0].Host
	if len(hosts) != 2 {
		t.Fatalf("expected 2 hosts (bare + www), got %d", len(hosts))
	}
	if hosts[0] != "myapp.com" || hosts[1] != "www.myapp.com" {
		t.Errorf("domain hosts = %v", hosts)
	}
}

func TestBuildFullConfig_SPAHandlers(t *testing.T) {
	b := newTestBuilder()

	deploys := []ActiveDeployment{
		{
			DeploymentID: "dpl_spa1",
			ProjectID:    "prj_spa",
			ProjectSlug:  "spa-app",
			Branch:       "main",
			IsProduction: true,
			ArtifactPath: "/app/deployments/prj_spa/dpl_spa1",
			Framework:    "vite",
		},
	}

	config := b.BuildFullConfig(deploys, nil)
	routes := config.Apps.HTTP.Servers["main"].Routes

	var prodRoute *CaddyRoute
	for i := range routes {
		if routes[i].ID == "route-prod-prj_spa" {
			prodRoute = &routes[i]
			break
		}
	}

	if prodRoute == nil {
		t.Fatal("missing production route")
	}

	// SPA routes should have subroute handler (3rd handler: encode, headers, subroute)
	if len(prodRoute.Handle) < 3 {
		t.Fatalf("expected at least 3 handlers for SPA, got %d", len(prodRoute.Handle))
	}

	subroute := prodRoute.Handle[2]
	if subroute.Handler != "subroute" {
		t.Errorf("expected subroute handler, got %q", subroute.Handler)
	}
}

func TestBuildFullConfig_StaticHandlers(t *testing.T) {
	b := newTestBuilder()

	deploys := []ActiveDeployment{
		{
			DeploymentID: "dpl_static1",
			ProjectID:    "prj_static",
			ProjectSlug:  "static-app",
			Branch:       "main",
			IsProduction: true,
			ArtifactPath: "/app/deployments/prj_static/dpl_static1",
			Framework:    "hugo",
		},
	}

	config := b.BuildFullConfig(deploys, nil)
	routes := config.Apps.HTTP.Servers["main"].Routes

	var prodRoute *CaddyRoute
	for i := range routes {
		if routes[i].ID == "route-prod-prj_static" {
			prodRoute = &routes[i]
			break
		}
	}

	if prodRoute == nil {
		t.Fatal("missing production route")
	}

	// Static routes: encode + headers + file_server (no subroute)
	if len(prodRoute.Handle) != 3 {
		t.Fatalf("expected 3 handlers for static, got %d", len(prodRoute.Handle))
	}

	fs := prodRoute.Handle[2]
	if fs.Handler != "file_server" {
		t.Errorf("expected file_server handler, got %q", fs.Handler)
	}
}

func TestBuildFullConfig_TLSPolicies(t *testing.T) {
	b := NewConfigBuilder(BuilderConfig{
		PlatformDomain: "hostbox.example.com",
		ACMEEmail:      "admin@example.com",
		DNSProvider:    "cloudflare",
		DNSProviderConf: json.RawMessage(`{"name":"cloudflare","api_token":"{env.CF_API_TOKEN}"}`),
	})

	config := b.BuildFullConfig(nil, nil)
	policies := config.Apps.TLS.Automation.Policies

	if len(policies) != 2 {
		t.Fatalf("expected 2 TLS policies (platform + wildcard), got %d", len(policies))
	}

	if policies[0].Subjects[0] != "hostbox.example.com" {
		t.Errorf("policy 0 subject = %q", policies[0].Subjects[0])
	}
	if policies[1].Subjects[0] != "*.hostbox.example.com" {
		t.Errorf("policy 1 subject = %q", policies[1].Subjects[0])
	}
	if policies[1].Issuers[0].Challenges == nil {
		t.Error("wildcard policy should have DNS challenge")
	}
}

func TestBuildFullConfig_TLSNoDNS(t *testing.T) {
	b := NewConfigBuilder(BuilderConfig{
		PlatformDomain: "hostbox.example.com",
		ACMEEmail:      "admin@example.com",
	})

	config := b.BuildFullConfig(nil, nil)
	policies := config.Apps.TLS.Automation.Policies

	if len(policies) != 1 {
		t.Fatalf("expected 1 TLS policy (no DNS), got %d", len(policies))
	}
}

func TestBuildFullConfig_JSONMarshal(t *testing.T) {
	b := newTestBuilder()
	config := b.BuildFullConfig(nil, nil)

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}

	// Verify it roundtrips
	var decoded CaddyConfig
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to unmarshal config: %v", err)
	}

	if decoded.Admin.Listen != "localhost:2019" {
		t.Error("roundtrip failed for admin listen")
	}
}
