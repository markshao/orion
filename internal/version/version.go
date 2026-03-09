package version

var (
	// Version is the current version of Orion.
	// This should be set via ldflags during build.
	Version = "v0.0.9"

	// Commit is the git commit hash of the build.
	// This should be set via ldflags during build.
	Commit = "none"

	// Date is the build date.
	// This should be set via ldflags during build.
	Date = "unknown"
)
