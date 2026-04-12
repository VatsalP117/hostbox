package caddy

import "testing"

func TestIsSPAFramework(t *testing.T) {
	spa := []string{"nextjs", "vite", "cra", "svelte", "angular", "vue", "sveltekit"}
	for _, f := range spa {
		if !isSPAFramework(f) {
			t.Errorf("expected %q to be SPA", f)
		}
	}

	notSPA := []string{"astro", "gatsby", "nuxt", "hugo", "html", "static", ""}
	for _, f := range notSPA {
		if isSPAFramework(f) {
			t.Errorf("expected %q to not be SPA", f)
		}
	}

	// Unknown defaults to SPA
	if !isSPAFramework("unknown-framework") {
		t.Error("unknown frameworks should default to SPA")
	}
}

func TestSlugify(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"feature/login", "feature-login"},
		{"Main", "main"},
		{"feat: add tests", "feat-add-tests"},
		{"Hello World!", "hello-world"},
		{"my--app", "my-app"},
		{" leading-trailing ", "leading-trailing"},
		{"", "default"},
		{"release/v1.2.3", "release-v1-2-3"},
	}

	for _, tt := range tests {
		got := Slugify(tt.input)
		if got != tt.want {
			t.Errorf("Slugify(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestGroupByProject(t *testing.T) {
	deployments := []ActiveDeployment{
		{DeploymentID: "d1", ProjectID: "p1"},
		{DeploymentID: "d2", ProjectID: "p1"},
		{DeploymentID: "d3", ProjectID: "p2"},
	}

	result := groupByProject(deployments)
	if len(result["p1"]) != 2 {
		t.Errorf("expected 2 deployments for p1, got %d", len(result["p1"]))
	}
	if len(result["p2"]) != 1 {
		t.Errorf("expected 1 deployment for p2, got %d", len(result["p2"]))
	}
}
