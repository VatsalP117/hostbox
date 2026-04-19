package detect

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// Framework holds the detected build configuration for a project.
type Framework struct {
	Name            string // "nextjs", "vite", "cra", "astro", "gatsby", "nuxt", "sveltekit", "hugo", "static", "unknown"
	DisplayName     string // "Next.js", "Vite", etc.
	BuildCommand    string // "npm run build", "npm run generate", etc.
	OutputDirectory string // "out", "dist", "build", "public", ".output/public", "."
	ServingMode     string // "spa" or "static"
}

// PackageJSON represents the subset of package.json fields we need.
type PackageJSON struct {
	Name            string            `json:"name"`
	Dependencies    map[string]string `json:"dependencies"`
	DevDependencies map[string]string `json:"devDependencies"`
	Scripts         map[string]string `json:"scripts"`
	Workspaces      json.RawMessage   `json:"workspaces"`
	Engines         struct {
		Node string `json:"node"`
	} `json:"engines"`
}

var errFrameworkNotDetected = errors.New("framework not detected")

// knownFrameworks is checked in priority order (first match wins).
var knownFrameworks = []struct {
	Dep       string
	DevDep    bool
	Framework Framework
}{
	{
		Dep: "next",
		Framework: Framework{
			Name:            "nextjs",
			DisplayName:     "Next.js",
			BuildCommand:    "npm run build",
			OutputDirectory: "out",
			ServingMode:     "static",
		},
	},
	{
		Dep: "react-scripts",
		Framework: Framework{
			Name:            "cra",
			DisplayName:     "Create React App",
			BuildCommand:    "npm run build",
			OutputDirectory: "build",
			ServingMode:     "spa",
		},
	},
	{
		Dep:    "vite",
		DevDep: true,
		Framework: Framework{
			Name:            "vite",
			DisplayName:     "Vite",
			BuildCommand:    "npm run build",
			OutputDirectory: "dist",
			ServingMode:     "spa",
		},
	},
	{
		Dep: "astro",
		Framework: Framework{
			Name:            "astro",
			DisplayName:     "Astro",
			BuildCommand:    "npm run build",
			OutputDirectory: "dist",
			ServingMode:     "static",
		},
	},
	{
		Dep: "gatsby",
		Framework: Framework{
			Name:            "gatsby",
			DisplayName:     "Gatsby",
			BuildCommand:    "npm run build",
			OutputDirectory: "public",
			ServingMode:     "static",
		},
	},
	{
		Dep: "nuxt",
		Framework: Framework{
			Name:            "nuxt",
			DisplayName:     "Nuxt 3",
			BuildCommand:    "npm run generate",
			OutputDirectory: ".output/public",
			ServingMode:     "static",
		},
	},
	{
		Dep: "@sveltejs/kit",
		Framework: Framework{
			Name:            "sveltekit",
			DisplayName:     "SvelteKit",
			BuildCommand:    "npm run build",
			OutputDirectory: "build",
			ServingMode:     "spa",
		},
	},
}

// DetectFramework reads package.json from sourceDir and returns the detected framework.
func DetectFramework(sourceDir string) (Framework, *PackageJSON, error) {
	// Check for Hugo first (no package.json needed)
	if isHugoProject(sourceDir) {
		return Framework{
			Name:            "hugo",
			DisplayName:     "Hugo",
			BuildCommand:    "hugo --minify",
			OutputDirectory: "public",
			ServingMode:     "static",
		}, nil, nil
	}

	pkg, err := readPackageJSON(sourceDir)
	if err != nil {
		// No package.json → check for index.html (plain static site)
		if errors.Is(err, os.ErrNotExist) {
			if _, err2 := os.Stat(filepath.Join(sourceDir, "index.html")); err2 == nil {
				return Framework{
					Name:            "static",
					DisplayName:     "Static HTML",
					BuildCommand:    "",
					OutputDirectory: ".",
					ServingMode:     "static",
				}, nil, nil
			}
			return Framework{}, nil, fmt.Errorf("no package.json or index.html found: %w", err)
		}
		return Framework{}, nil, err
	}

	if fw, err := detectKnownFramework(sourceDir, pkg); err == nil {
		return fw, pkg, nil
	} else if !errors.Is(err, errFrameworkNotDetected) {
		return Framework{}, nil, err
	}

	if IsWorkspaceProject(sourceDir, pkg) {
		if fw, err := detectWorkspaceFramework(sourceDir, pkg); err == nil {
			return fw, pkg, nil
		} else if !errors.Is(err, errFrameworkNotDetected) {
			return Framework{}, nil, err
		}
	}

	// Fallback: unknown Node.js project
	return Framework{
		Name:            "unknown",
		DisplayName:     "Node.js",
		BuildCommand:    "npm run build",
		OutputDirectory: "dist",
		ServingMode:     "spa",
	}, pkg, nil
}

func readPackageJSON(dir string) (*PackageJSON, error) {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return nil, err
	}

	var pkg PackageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, fmt.Errorf("invalid package.json: %w", err)
	}

	return &pkg, nil
}

func detectKnownFramework(sourceDir string, pkg *PackageJSON) (Framework, error) {
	allDeps := make(map[string]string)
	for k, v := range pkg.Dependencies {
		allDeps[k] = v
	}
	for k, v := range pkg.DevDependencies {
		allDeps[k] = v
	}

	for _, kf := range knownFrameworks {
		if kf.DevDep {
			if _, ok := pkg.DevDependencies[kf.Dep]; ok {
				return finalizeDetectedFramework(sourceDir, pkg, kf.Framework)
			}
		} else {
			if _, ok := allDeps[kf.Dep]; ok {
				return finalizeDetectedFramework(sourceDir, pkg, kf.Framework)
			}
		}
	}

	return Framework{}, errFrameworkNotDetected
}

func detectWorkspaceFramework(sourceDir string, rootPkg *PackageJSON) (Framework, error) {
	type candidate struct {
		relDir string
		pkg    *PackageJSON
		fw     Framework
	}

	var candidates []candidate

	err := filepath.WalkDir(sourceDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() != "package.json" {
			return nil
		}

		dir := filepath.Dir(path)
		if dir == sourceDir {
			return nil
		}

		pkg, err := readPackageJSON(dir)
		if err != nil {
			return err
		}
		fw, err := detectKnownFramework(dir, pkg)
		if errors.Is(err, errFrameworkNotDetected) {
			return nil
		}
		if err != nil {
			return err
		}

		relDir, err := filepath.Rel(sourceDir, dir)
		if err != nil {
			return err
		}

		candidates = append(candidates, candidate{
			relDir: filepath.ToSlash(relDir),
			pkg:    pkg,
			fw:     fw,
		})
		return nil
	})
	if err != nil {
		return Framework{}, err
	}

	if len(candidates) == 0 {
		return Framework{}, errFrameworkNotDetected
	}
	if len(candidates) > 1 {
		paths := make([]string, 0, len(candidates))
		for _, c := range candidates {
			paths = append(paths, c.relDir)
		}
		return Framework{}, fmt.Errorf("multiple workspace apps detected (%s); set the project root directory explicitly", strings.Join(paths, ", "))
	}

	selected := candidates[0]
	fw := selected.fw
	fw.DisplayName = fmt.Sprintf("%s (%s)", fw.DisplayName, selected.relDir)
	fw.OutputDirectory = filepath.ToSlash(filepath.Join(selected.relDir, fw.OutputDirectory))

	if rootPkg != nil && rootPkg.Scripts["build"] != "" {
		fw.BuildCommand = "npm run build"
	}

	return fw, nil
}

func finalizeDetectedFramework(sourceDir string, pkg *PackageJSON, fw Framework) (Framework, error) {
	if fw.Name != "nextjs" {
		return fw, nil
	}

	nextOutput := detectNextOutputMode(sourceDir)
	buildScript := ""
	if pkg != nil {
		buildScript = pkg.Scripts["build"]
	}

	if nextOutput == "standalone" {
		return Framework{}, fmt.Errorf("Next.js standalone output is not supported for static deployment; use output: \"export\" or configure custom build/output settings")
	}
	if nextOutput == "export" || strings.Contains(buildScript, "next export") {
		fw.OutputDirectory = "out"
		return fw, nil
	}

	return Framework{}, fmt.Errorf("Next.js projects must produce a static export (out/); configure output: \"export\" in next.config or set custom build/output settings")
}

func detectNextOutputMode(sourceDir string) string {
	matches, _ := filepath.Glob(filepath.Join(sourceDir, "next.config.*"))
	if len(matches) == 0 {
		return ""
	}

	re := regexp.MustCompile(`output\s*:\s*["'](standalone|export)["']`)
	for _, match := range matches {
		data, err := os.ReadFile(match)
		if err != nil {
			continue
		}
		found := re.FindStringSubmatch(string(data))
		if len(found) == 2 {
			return found[1]
		}
	}

	return ""
}

// isHugoProject checks for Hugo config files.
func isHugoProject(dir string) bool {
	hugoConfigs := []string{"hugo.toml", "hugo.yaml", "hugo.json", "config.toml", "config.yaml", "config.json"}
	for _, f := range hugoConfigs {
		if _, err := os.Stat(filepath.Join(dir, f)); err == nil {
			return true
		}
	}
	return false
}

// DetectNodeVersion extracts the Node.js version from package.json engines.node.
// Supports: "20", "20.x", ">=18", "^20.0.0", "~18.17.0"
func DetectNodeVersion(pkg *PackageJSON, defaultVersion string) string {
	if pkg == nil || pkg.Engines.Node == "" {
		return defaultVersion
	}

	re := regexp.MustCompile(`(\d+)`)
	match := re.FindString(pkg.Engines.Node)
	if match == "" {
		return defaultVersion
	}

	major, err := strconv.Atoi(match)
	if err != nil || major < 16 || major > 22 {
		return defaultVersion
	}

	return match
}

// ApplyOverrides merges project-level overrides into the detected framework config.
func ApplyOverrides(fw Framework, buildCmd, installCmd, outputDir string) Framework {
	if buildCmd != "" {
		fw.BuildCommand = buildCmd
	}
	if outputDir != "" {
		fw.OutputDirectory = outputDir
	}
	return fw
}

// AdaptCommandForPackageManager rewrites npm script commands to match the detected package manager.
func AdaptCommandForPackageManager(cmd, packageManager string) string {
	cmd = strings.TrimSpace(cmd)
	script, ok := strings.CutPrefix(cmd, "npm run ")
	if !ok {
		return cmd
	}

	switch packageManager {
	case "pnpm":
		return "pnpm run " + script
	case "yarn":
		return "yarn run " + script
	case "bun":
		return "bun run " + script
	default:
		return cmd
	}
}

// IsWorkspaceProject reports whether the project is a JavaScript workspace/monorepo root.
func IsWorkspaceProject(sourceDir string, pkg *PackageJSON) bool {
	if pkg != nil && len(pkg.Workspaces) > 0 && string(pkg.Workspaces) != "null" {
		return true
	}

	_, err := os.Stat(filepath.Join(sourceDir, "pnpm-workspace.yaml"))
	return err == nil
}
