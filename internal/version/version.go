package version

// These variables are set at build time using -ldflags
var (
	Version   = "dev"     // Set via -ldflags "-X github.com/tranmh/gassigeher/internal/version.Version=1.0"
	GitCommit = "unknown" // Set via -ldflags "-X github.com/tranmh/gassigeher/internal/version.GitCommit=$(git rev-parse --short HEAD)"
	BuildTime = "unknown" // Set via -ldflags "-X github.com/tranmh/gassigeher/internal/version.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
)

// Info returns version information
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildTime string `json:"build_time,omitempty"`
}

// Get returns the current version info
func Get() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildTime: BuildTime,
	}
}
