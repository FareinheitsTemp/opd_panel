package versions

import "context"

// Provider knows how to list and download a specific server type.
type Provider interface {
	Name() string
	GetVersions(ctx context.Context) ([]string, error)
	GetDownloadURL(ctx context.Context, version string) (string, error)
	// GetChecksum returns SHA256 hex of the jar, empty string if unavailable.
	GetChecksum(ctx context.Context, version string) (string, error)
}
