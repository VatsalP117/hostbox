package version

// Set at build time via ldflags:
//
//	go build -ldflags "-X github.com/VatsalP117/hostbox/internal/version.Version=1.0.0"
var (
	Version   = "dev"
	GitCommit = "unknown"
	BuildTime = "unknown"
)
