package hostnames

import "testing"

func TestNormalizeProjectSlug(t *testing.T) {
	got := NormalizeProjectSlug("My Extremely Long Project Name That Needs To Stay Within DNS Label Limits")
	if len(got) > MaxProjectSlugLength {
		t.Fatalf("slug length = %d, want <= %d", len(got), MaxProjectSlugLength)
	}
	if got == "" {
		t.Fatal("expected non-empty slug")
	}
}

func TestPreviewHostUsesDeploymentID(t *testing.T) {
	host := PreviewHost("my-app", "deploy_ABC123456789", "example.com")
	if host == "my-app-deploy-a.example.com" {
		t.Fatalf("host should preserve more entropy than %q, got %q", "my-app-deploy-a.example.com", host)
	}
	if len(host) > len("my-app-.example.com")+PreviewSuffixLength {
		t.Fatalf("host suffix too long: %q", host)
	}
}

func TestBranchHostTruncatesToSingleLabel(t *testing.T) {
	host := BranchHost(
		"project-name-that-is-still-pretty-long-for-preview-hosts",
		"feature/a-very-very-very-very-very-very-long-branch-name",
		"example.com",
	)

	label := host[:len(host)-len(".example.com")]
	if len(label) > MaxDNSLabelLength {
		t.Fatalf("label length = %d, want <= %d", len(label), MaxDNSLabelLength)
	}
}

func TestBranchHostKeepsProjectsDistinctWithLongBranchNames(t *testing.T) {
	branch := "feature/this-is-a-very-very-very-very-very-very-very-very-long-branch-name"
	hostA := BranchHost("apple-platform-project-with-a-fairly-long-name", branch, "example.com")
	hostB := BranchHost("avocado-platform-project-with-a-fairly-long-name", branch, "example.com")

	if hostA == hostB {
		t.Fatalf("expected distinct hosts, got %q", hostA)
	}
}

func TestReservedProjectLabel(t *testing.T) {
	label, ok := ReservedProjectLabel("example.com", "hostbox.example.com")
	if !ok {
		t.Fatal("expected reserved label")
	}
	if label != "hostbox" {
		t.Fatalf("label = %q, want hostbox", label)
	}
}

func TestReservedProjectLabelSkipsNestedDashboardDomain(t *testing.T) {
	if _, ok := ReservedProjectLabel("example.com", "ops.hostbox.example.com"); ok {
		t.Fatal("expected nested dashboard domain to be ignored")
	}
}
