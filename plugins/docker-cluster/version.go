package dockercluster

// Version information is injected by the Dockerfile using the package path in
// the cloned CoreDNS module: github.com/coredns/coredns/plugin/docker-cluster.
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
