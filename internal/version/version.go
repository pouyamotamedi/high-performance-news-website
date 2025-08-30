package version

import (
	"fmt"
	"runtime"
)

const (
	// Version represents the current version of the application
	Version = "1.3.0"
	
	// BuildDate will be set during build time
	BuildDate = "development"
	
	// GitCommit will be set during build time
	GitCommit = "development"
	
	// GitBranch will be set during build time
	GitBranch = "main"
)

// Info contains version information
type Info struct {
	Version   string `json:"version"`
	BuildDate string `json:"build_date"`
	GitCommit string `json:"git_commit"`
	GitBranch string `json:"git_branch"`
	GoVersion string `json:"go_version"`
	Platform  string `json:"platform"`
}

// Get returns version information
func Get() Info {
	return Info{
		Version:   Version,
		BuildDate: BuildDate,
		GitCommit: GitCommit,
		GitBranch: GitBranch,
		GoVersion: runtime.Version(),
		Platform:  fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH),
	}
}

// String returns a formatted version string
func (i Info) String() string {
	return fmt.Sprintf("Version: %s, Build Date: %s, Git Commit: %s, Go Version: %s, Platform: %s",
		i.Version, i.BuildDate, i.GitCommit, i.GoVersion, i.Platform)
}

// GetVersion returns just the version string
func GetVersion() string {
	return Version
}

// GetFullVersion returns a detailed version string
func GetFullVersion() string {
	info := Get()
	return info.String()
}