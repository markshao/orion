package version

var (
	// Version is the current version of DevSwarm.
	// This should be set via ldflags during build.
	Version = "dev"

	// Commit is the git commit hash of the build.
	// This should be set via ldflags during build.
	Commit = "none"

	// Date is the build date.
	// This should be set via ldflags during build.
	Date = "unknown"
)
