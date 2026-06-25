package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const vanillaManifestURL = "https://piston-meta.mojang.com/mc/game/version_manifest_v2.json"

type VanillaProvider struct{ client *http.Client }

func NewVanillaProvider() *VanillaProvider {
	return &VanillaProvider{client: &http.Client{}}
}

func (p *VanillaProvider) Name() string { return "vanilla" }

func (p *VanillaProvider) GetVersions(ctx context.Context) ([]string, error) {
	manifest, err := p.fetchManifest(ctx)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, v := range manifest.Versions {
		if v.Type == "release" {
			out = append(out, v.ID)
		}
	}
	return out, nil
}

func (p *VanillaProvider) GetDownloadURL(ctx context.Context, version string) (string, error) {
	manifest, err := p.fetchManifest(ctx)
	if err != nil {
		return "", err
	}
	for _, v := range manifest.Versions {
		if v.ID == version {
			// Fetch version meta
			var meta struct {
				Downloads struct {
					Server struct {
						URL  string `json:"url"`
						SHA1 string `json:"sha1"`
					} `json:"server"`
				} `json:"downloads"`
			}
			req, _ := http.NewRequestWithContext(ctx, http.MethodGet, v.URL, nil)
			resp, err := p.client.Do(req)
			if err != nil {
				return "", err
			}
			defer resp.Body.Close()
			if err := json.NewDecoder(resp.Body).Decode(&meta); err != nil {
				return "", err
			}
			return meta.Downloads.Server.URL, nil
		}
	}
	return "", fmt.Errorf("vanilla version %q not found", version)
}

func (p *VanillaProvider) GetChecksum(_ context.Context, _ string) (string, error) {
	// SHA1 from manifest, but we verify SHA256 in downloader
	// Vanilla only provides SHA1 — skip SHA256 check
	return "", nil
}

type vanillaManifest struct {
	Versions []struct {
		ID   string `json:"id"`
		Type string `json:"type"`
		URL  string `json:"url"`
	} `json:"versions"`
}

func (p *VanillaProvider) fetchManifest(ctx context.Context) (*vanillaManifest, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, vanillaManifestURL, nil)
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var m vanillaManifest
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return nil, err
	}
	return &m, nil
}
