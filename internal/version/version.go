package version

var (
	// Version is the current version of Orion.
	// Defaults to "dev" for local/source builds and is overridden via ldflags in release builds.
	Version = "dev"

	// Commit is the git commit hash of the build.
	// This should be set via ldflags during build.
	Commit = "none"

	// Date is the build date.
	// This should be set via ldflags during build.
	Date = "unknown"
)
