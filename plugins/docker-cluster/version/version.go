// Package version contains build-time version information injected via ldflags.
package version

// Version information set at build time via ldflags.
// Example: go build -ldflags="-X 'github.com/traefikturkey/joyride/plugins/docker-cluster/version.Version=1.0.0'"
var (
	// Version is the semantic version (e.g., "1.0.0")
	Version = "dev"

	// GitCommit is the git commit SHA
	GitCommit = "unknown"

	// BuildTime is the build timestamp in ISO 8601 format
	BuildTime = "unknown"
)

// Info holds version information for JSON serialization.
type Info struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildTime string `json:"build_time"`
}

// GetInfo returns the current version information.
func GetInfo() Info {
	return Info{
		Version:   Version,
		GitCommit: GitCommit,
		BuildTime: BuildTime,
	}
}
