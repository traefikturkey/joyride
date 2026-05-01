// Package version contains build-time version information injected via ldflags.
package version

// Version information set at build time via ldflags.
// At build time this package's import path is
// github.com/coredns/coredns/plugin/docker-cluster/version
// because the plugin source is copied into the cloned CoreDNS module tree.
// See the Dockerfile builder stage for the matching -X flags.
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
