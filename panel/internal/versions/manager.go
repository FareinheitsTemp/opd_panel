package versions

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/FareinheitsTemp/opd_panel/panel/internal/domain"
)

type Manager struct {
	providers map[domain.ServerType]Provider
	cache     *Cache
	downloader *Downloader
}

func NewManager(cacheDir string) *Manager {
	m := &Manager{
		providers:  make(map[domain.ServerType]Provider),
		cache:      NewCache(cacheDir),
		downloader: NewDownloader(),
	}
	m.register()
	return m
}

func (m *Manager) register() {
	// providers registered here once implemented
	// m.providers[domain.ServerTypePaper] = providers.NewPaperProvider()
	// m.providers[domain.ServerTypeVanilla] = providers.NewVanillaProvider()
}

// Resolve ensures the jar for (serverType, version) exists in destDir/server.jar.
func (m *Manager) Resolve(ctx context.Context, t domain.ServerType, version, destDir string) (string, error) {
	jarDest := filepath.Join(destDir, "server.jar")

	// 1. Check local cache
	if cached, err := m.cache.Get(t, version); err == nil {
		return jarDest, copyFile(cached, jarDest)
	}

	// 2. Find provider
	provider, ok := m.providers[t]
	if !ok {
		return "", fmt.Errorf("no provider for server type %q", t)
	}

	// 3. Get download URL
	url, err := provider.GetDownloadURL(ctx, version)
	if err != nil {
		return "", fmt.Errorf("get download url: %w", err)
	}

	checksum, _ := provider.GetChecksum(ctx, version)

	// 4. Download
	tmpPath, err := m.downloader.Download(ctx, url, checksum)
	if err != nil {
		return "", fmt.Errorf("download: %w", err)
	}

	// 5. Store in cache
	if err := m.cache.Store(t, version, tmpPath); err != nil {
		// non-fatal: cache miss is recoverable
		_ = err
	}

	// 6. Copy to server dir
	return jarDest, copyFile(tmpPath, jarDest)
}

func (m *Manager) ListVersions(ctx context.Context, t domain.ServerType) ([]string, error) {
	p, ok := m.providers[t]
	if !ok {
		return nil, fmt.Errorf("no provider for %q", t)
	}
	return p.GetVersions(ctx)
}

func copyFile(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
