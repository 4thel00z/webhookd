package buildinfo

// Version is set at build-time via -ldflags, e.g.
//
//	-X webhookd/internal/buildinfo.Version=v1.2.3
//
// Defaults to "dev" for local builds.
var Version = "dev"
