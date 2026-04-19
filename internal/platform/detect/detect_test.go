package detect

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- Framework Detection Tests ---

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	err := os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDetectFramework_NextJS(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"next":"14.0.0","react":"18.0.0"}}`)
	writeFile(t, dir, "next.config.js", `module.exports = { output: "export" }`)

	fw, pkg, err := DetectFramework(dir)
	if err != nil {
		t.Fatal(err)
	}
	if fw.Name != "nextjs" {
		t.Errorf("expected nextjs, got %s", fw.Name)
	}
	if fw.OutputDirectory != "out" {
		t.Errorf("expected out, got %s", fw.OutputDirectory)
	}
	if pkg == nil {
		t.Error("expected package.json to be parsed")
	}
}

func TestDetectFramework_CRA(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"react":"18.0.0","react-scripts":"5.0.0"}}`)

	fw, _, err := DetectFramework(dir)
	if err != nil {
		t.Fatal(err)
	}
	if fw.Name != "cra" {
		t.Errorf("expected cra, got %s", fw.Name)
	}
	if fw.OutputDirectory != "build" {
		t.Errorf("expected build, got %s", fw.OutputDirectory)
	}
}

func TestDetectFramework_Vite(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"react":"18.0.0"},"devDependencies":{"vite":"5.0.0"}}`)

	fw, _, err := DetectFramework(dir)
	if err != nil {
		t.Fatal(err)
	}
	if fw.Name != "vite" {
		t.Errorf("expected vite, got %s", fw.Name)
	}
	if fw.OutputDirectory != "dist" {
		t.Errorf("expected dist, got %s", fw.OutputDirectory)
	}
}

func TestDetectFramework_ViteInDependencies_NotDetected(t *testing.T) {
	// Vite should only be detected in devDependencies
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"vite":"5.0.0"}}`)

	fw, _, err := DetectFramework(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Vite in regular deps should not match since DevDep=true
	if fw.Name != "unknown" {
		t.Errorf("expected unknown (vite only checked in devDeps), got %s", fw.Name)
	}
}

func TestDetectFramework_Astro(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"astro":"4.0.0"}}`)

	fw, _, err := DetectFramework(dir)
	if err != nil {
		t.Fatal(err)
	}
	if fw.Name != "astro" {
		t.Errorf("expected astro, got %s", fw.Name)
	}
}

func TestDetectFramework_Gatsby(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"gatsby":"5.0.0"}}`)

	fw, _, err := DetectFramework(dir)
	if err != nil {
		t.Fatal(err)
	}
	if fw.Name != "gatsby" {
		t.Errorf("expected gatsby, got %s", fw.Name)
	}
	if fw.OutputDirectory != "public" {
		t.Errorf("expected public, got %s", fw.OutputDirectory)
	}
}

func TestDetectFramework_Nuxt(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"nuxt":"3.0.0"}}`)

	fw, _, err := DetectFramework(dir)
	if err != nil {
		t.Fatal(err)
	}
	if fw.Name != "nuxt" {
		t.Errorf("expected nuxt, got %s", fw.Name)
	}
	if fw.BuildCommand != "npm run generate" {
		t.Errorf("expected 'npm run generate', got %s", fw.BuildCommand)
	}
}

func TestDetectFramework_SvelteKit(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"@sveltejs/kit":"2.0.0"}}`)

	fw, _, err := DetectFramework(dir)
	if err != nil {
		t.Fatal(err)
	}
	if fw.Name != "sveltekit" {
		t.Errorf("expected sveltekit, got %s", fw.Name)
	}
}

func TestDetectFramework_Hugo(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "hugo.toml", `baseURL = "https://example.com"`)

	fw, pkg, err := DetectFramework(dir)
	if err != nil {
		t.Fatal(err)
	}
	if fw.Name != "hugo" {
		t.Errorf("expected hugo, got %s", fw.Name)
	}
	if fw.BuildCommand != "hugo --minify" {
		t.Errorf("expected 'hugo --minify', got %s", fw.BuildCommand)
	}
	if pkg != nil {
		t.Error("expected nil package.json for hugo")
	}
}

func TestDetectFramework_Hugo_ConfigYaml(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "config.yaml", `baseURL: https://example.com`)

	fw, _, err := DetectFramework(dir)
	if err != nil {
		t.Fatal(err)
	}
	if fw.Name != "hugo" {
		t.Errorf("expected hugo, got %s", fw.Name)
	}
}

func TestDetectFramework_StaticHTML(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "index.html", `<html><body>Hello</body></html>`)

	fw, pkg, err := DetectFramework(dir)
	if err != nil {
		t.Fatal(err)
	}
	if fw.Name != "static" {
		t.Errorf("expected static, got %s", fw.Name)
	}
	if fw.BuildCommand != "" {
		t.Errorf("expected empty build command, got %s", fw.BuildCommand)
	}
	if fw.OutputDirectory != "." {
		t.Errorf("expected '.', got %s", fw.OutputDirectory)
	}
	if pkg != nil {
		t.Error("expected nil package.json for static")
	}
}

func TestDetectFramework_EmptyDir_Error(t *testing.T) {
	dir := t.TempDir()

	_, _, err := DetectFramework(dir)
	if err == nil {
		t.Error("expected error for empty directory")
	}
}

func TestDetectFramework_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{invalid json}`)

	_, _, err := DetectFramework(dir)
	if err == nil {
		t.Error("expected error for invalid package.json")
	}
}

func TestDetectFramework_UnknownProject(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"express":"4.0.0"}}`)

	fw, _, err := DetectFramework(dir)
	if err != nil {
		t.Fatal(err)
	}
	if fw.Name != "unknown" {
		t.Errorf("expected unknown, got %s", fw.Name)
	}
}

func TestDetectFramework_NextJSPriorityOverReact(t *testing.T) {
	// Next.js projects also have react — Next should be detected, not CRA
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"next":"14.0.0","react":"18.0.0","react-scripts":"5.0.0"}}`)
	writeFile(t, dir, "next.config.js", `module.exports = { output: "export" }`)

	fw, _, err := DetectFramework(dir)
	if err != nil {
		t.Fatal(err)
	}
	if fw.Name != "nextjs" {
		t.Errorf("expected nextjs (higher priority), got %s", fw.Name)
	}
}

func TestDetectFramework_NextJSStandaloneUnsupported(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{"dependencies":{"next":"16.0.0","react":"19.0.0"}}`)
	writeFile(t, dir, "next.config.ts", `export default { output: "standalone" }`)

	_, _, err := DetectFramework(dir)
	if err == nil {
		t.Fatal("expected standalone Next.js to be rejected")
	}
	if !strings.Contains(err.Error(), `output: "export"`) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestDetectFramework_WorkspaceNextExport(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{
		"private": true,
		"scripts": {"build": "turbo build"},
		"devDependencies": {"turbo": "^2.0.0"}
	}`)
	writeFile(t, dir, "pnpm-workspace.yaml", "packages:\n  - apps/*\n")
	writeFile(t, dir, "apps/web/package.json", `{
		"name": "web",
		"scripts": {"build": "next build"},
		"dependencies": {"next": "16.0.0", "react": "19.0.0"}
	}`)
	writeFile(t, dir, "apps/web/next.config.ts", `export default { output: "export" }`)

	fw, pkg, err := DetectFramework(dir)
	if err != nil {
		t.Fatalf("DetectFramework() error: %v", err)
	}
	if pkg == nil {
		t.Fatal("expected root package.json to be returned")
	}
	if fw.Name != "nextjs" {
		t.Fatalf("expected nextjs, got %s", fw.Name)
	}
	if fw.BuildCommand != "npm run build" {
		t.Fatalf("expected root build command, got %q", fw.BuildCommand)
	}
	if fw.OutputDirectory != "apps/web/out" {
		t.Fatalf("expected workspace output path, got %q", fw.OutputDirectory)
	}
}

func TestDetectFramework_WorkspaceNextStandaloneUnsupported(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{
		"private": true,
		"scripts": {"build": "turbo build"},
		"devDependencies": {"turbo": "^2.0.0"}
	}`)
	writeFile(t, dir, "pnpm-workspace.yaml", "packages:\n  - apps/*\n")
	writeFile(t, dir, "apps/web/package.json", `{
		"name": "web",
		"scripts": {"build": "next build"},
		"dependencies": {"next": "16.0.0", "react": "19.0.0"}
	}`)
	writeFile(t, dir, "apps/web/next.config.ts", `export default { output: "standalone" }`)

	_, _, err := DetectFramework(dir)
	if err == nil {
		t.Fatal("expected standalone workspace app to be rejected")
	}
	if !strings.Contains(err.Error(), "standalone") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Node Version Detection Tests ---

func TestDetectNodeVersion_FromEngines(t *testing.T) {
	tests := []struct {
		name     string
		engines  string
		expected string
	}{
		{"exact major", "20", "20"},
		{"major.x", "20.x", "20"},
		{"semver", "20.0.0", "20"},
		{"caret", "^20.0.0", "20"},
		{"tilde", "~18.17.0", "18"},
		{"gte", ">=18", "18"},
		{"range", ">=18 <21", "18"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pkg := &PackageJSON{}
			pkg.Engines.Node = tt.engines
			v := DetectNodeVersion(pkg, "20")
			if v != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, v)
			}
		})
	}
}

func TestDetectNodeVersion_NilPkg(t *testing.T) {
	v := DetectNodeVersion(nil, "20")
	if v != "20" {
		t.Errorf("expected default 20, got %s", v)
	}
}

func TestDetectNodeVersion_EmptyEngines(t *testing.T) {
	pkg := &PackageJSON{}
	v := DetectNodeVersion(pkg, "20")
	if v != "20" {
		t.Errorf("expected default 20, got %s", v)
	}
}

func TestDetectNodeVersion_TooOld(t *testing.T) {
	pkg := &PackageJSON{}
	pkg.Engines.Node = "14"
	v := DetectNodeVersion(pkg, "20")
	if v != "20" {
		t.Errorf("expected default 20 for old version, got %s", v)
	}
}

func TestDetectNodeVersion_TooNew(t *testing.T) {
	pkg := &PackageJSON{}
	pkg.Engines.Node = "30"
	v := DetectNodeVersion(pkg, "20")
	if v != "20" {
		t.Errorf("expected default 20 for too-new version, got %s", v)
	}
}

// --- Apply Overrides Tests ---

func TestApplyOverrides(t *testing.T) {
	fw := Framework{
		Name:            "vite",
		BuildCommand:    "npm run build",
		OutputDirectory: "dist",
	}

	result := ApplyOverrides(fw, "yarn build", "", "out")
	if result.BuildCommand != "yarn build" {
		t.Errorf("expected 'yarn build', got %s", result.BuildCommand)
	}
	if result.OutputDirectory != "out" {
		t.Errorf("expected 'out', got %s", result.OutputDirectory)
	}
}

func TestApplyOverrides_EmptyKeepsDefaults(t *testing.T) {
	fw := Framework{
		BuildCommand:    "npm run build",
		OutputDirectory: "dist",
	}

	result := ApplyOverrides(fw, "", "", "")
	if result.BuildCommand != "npm run build" {
		t.Errorf("expected original build command")
	}
	if result.OutputDirectory != "dist" {
		t.Errorf("expected original output directory")
	}
}

// --- Package Manager Detection Tests ---

func TestDetectPackageManager_PNPM(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pnpm-lock.yaml", "lockfileVersion: 6")

	pm := DetectPackageManager(dir)
	if pm.Name != "pnpm" {
		t.Errorf("expected pnpm, got %s", pm.Name)
	}
	if pm.InstallCommand != "pnpm install --frozen-lockfile" {
		t.Errorf("unexpected install command: %s", pm.InstallCommand)
	}
}

func TestDetectPackageManager_Yarn(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "yarn.lock", "# yarn lockfile v1")

	pm := DetectPackageManager(dir)
	if pm.Name != "yarn" {
		t.Errorf("expected yarn, got %s", pm.Name)
	}
}

func TestDetectPackageManager_Bun(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "bun.lockb", "binary-content")

	pm := DetectPackageManager(dir)
	if pm.Name != "bun" {
		t.Errorf("expected bun, got %s", pm.Name)
	}
}

func TestDetectPackageManager_NPM(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package-lock.json", `{"lockfileVersion": 3}`)

	pm := DetectPackageManager(dir)
	if pm.Name != "npm" {
		t.Errorf("expected npm, got %s", pm.Name)
	}
	if pm.InstallCommand != "npm ci" {
		t.Errorf("expected 'npm ci', got %s", pm.InstallCommand)
	}
}

func TestDetectPackageManager_Fallback(t *testing.T) {
	dir := t.TempDir()

	pm := DetectPackageManager(dir)
	if pm.Name != "npm" {
		t.Errorf("expected npm fallback, got %s", pm.Name)
	}
	if pm.InstallCommand != "npm install" {
		t.Errorf("expected 'npm install', got %s", pm.InstallCommand)
	}
	if pm.LockFile != "" {
		t.Errorf("expected empty lock file, got %s", pm.LockFile)
	}
}

func TestDetectPackageManager_PNPMPriorityOverNPM(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pnpm-lock.yaml", "lockfileVersion: 6")
	writeFile(t, dir, "package-lock.json", `{"lockfileVersion": 3}`)

	pm := DetectPackageManager(dir)
	if pm.Name != "pnpm" {
		t.Errorf("expected pnpm (higher priority), got %s", pm.Name)
	}
}

// --- Hash Lock File Tests ---

func TestHashLockFile(t *testing.T) {
	dir := t.TempDir()
	content := "some lock file content"
	writeFile(t, dir, "package-lock.json", content)

	hash := HashLockFile(dir, "package-lock.json")

	expected := sha256.Sum256([]byte(content))
	expectedHex := hex.EncodeToString(expected[:])

	if hash != expectedHex {
		t.Errorf("hash mismatch: expected %s, got %s", expectedHex, hash)
	}
}

func TestHashLockFile_EmptyFilename(t *testing.T) {
	dir := t.TempDir()
	hash := HashLockFile(dir, "")
	if hash != "" {
		t.Errorf("expected empty hash for empty filename, got %s", hash)
	}
}

func TestHashLockFile_MissingFile(t *testing.T) {
	dir := t.TempDir()
	hash := HashLockFile(dir, "nonexistent.lock")
	if hash != "" {
		t.Errorf("expected empty hash for missing file, got %s", hash)
	}
}

func TestAdaptCommandForPackageManager(t *testing.T) {
	tests := []struct {
		name     string
		cmd      string
		manager  string
		expected string
	}{
		{name: "npm unchanged", cmd: "npm run build", manager: "npm", expected: "npm run build"},
		{name: "pnpm build", cmd: "npm run build", manager: "pnpm", expected: "pnpm run build"},
		{name: "yarn generate", cmd: "npm run generate", manager: "yarn", expected: "yarn run generate"},
		{name: "bun build", cmd: "npm run build", manager: "bun", expected: "bun run build"},
		{name: "non npm command unchanged", cmd: "hugo --minify", manager: "pnpm", expected: "hugo --minify"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AdaptCommandForPackageManager(tt.cmd, tt.manager)
			if got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestIsWorkspaceProject_PackageJSONWorkspaces(t *testing.T) {
	dir := t.TempDir()
	raw := json.RawMessage(`["apps/*", "packages/*"]`)
	pkg := &PackageJSON{Workspaces: raw}

	if !IsWorkspaceProject(dir, pkg) {
		t.Fatal("expected workspace project to be detected from package.json")
	}
}

func TestIsWorkspaceProject_PNPMWorkspaceFile(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pnpm-workspace.yaml", "packages:\n  - apps/*\n")

	if !IsWorkspaceProject(dir, nil) {
		t.Fatal("expected workspace project to be detected from pnpm-workspace.yaml")
	}
}

func TestIsWorkspaceProject_FalseForRegularPackage(t *testing.T) {
	dir := t.TempDir()

	if IsWorkspaceProject(dir, &PackageJSON{}) {
		t.Fatal("expected regular package to not be treated as a workspace")
	}
}
