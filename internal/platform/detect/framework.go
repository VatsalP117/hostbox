package detect

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
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
	Engines         struct {
		Node string `json:"node"`
	} `json:"engines"`
}

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

	pkgPath := filepath.Join(sourceDir, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		// No package.json → check for index.html (plain static site)
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

	var pkg PackageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return Framework{}, nil, fmt.Errorf("invalid package.json: %w", err)
	}

	// Merge deps + devDeps for lookup
	allDeps := make(map[string]string)
	for k, v := range pkg.Dependencies {
		allDeps[k] = v
	}
	for k, v := range pkg.DevDependencies {
		allDeps[k] = v
	}

	// Check known frameworks in priority order
	for _, kf := range knownFrameworks {
		if kf.DevDep {
			if _, ok := pkg.DevDependencies[kf.Dep]; ok {
				return kf.Framework, &pkg, nil
			}
		} else {
			if _, ok := allDeps[kf.Dep]; ok {
				return kf.Framework, &pkg, nil
			}
		}
	}

	// Fallback: unknown Node.js project
	return Framework{
		Name:            "unknown",
		DisplayName:     "Node.js",
		BuildCommand:    "npm run build",
		OutputDirectory: "dist",
		ServingMode:     "spa",
	}, &pkg, nil
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
