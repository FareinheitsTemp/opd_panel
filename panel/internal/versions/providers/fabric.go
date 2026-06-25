package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const fabricMeta = "https://meta.fabricmc.net/v2"

type FabricProvider struct {
	client        *http.Client
	loaderVersion string // pinned loader version, empty = latest
}

func NewFabricProvider() *FabricProvider {
	return &FabricProvider{client: &http.Client{}}
}

func (p *FabricProvider) Name() string { return "fabric" }

func (p *FabricProvider) GetVersions(ctx context.Context) ([]string, error) {
	url := fabricMeta + "/versions/game"
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var data []struct {
		Version string `json:"version"`
		Stable  bool   `json:"stable"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	var out []string
	for _, v := range data {
		if v.Stable {
			out = append(out, v.Version)
		}
	}
	return out, nil
}

func (p *FabricProvider) GetDownloadURL(ctx context.Context, version string) (string, error) {
	loader, err := p.getLatestLoader(ctx)
	if err != nil {
		return "", err
	}
	// Direct server jar download endpoint
	return fmt.Sprintf("%s/versions/loader/%s/%s/1.0.1/server/jar", fabricMeta, version, loader), nil
}

func (p *FabricProvider) GetChecksum(_ context.Context, _ string) (string, error) {
	return "", nil // Fabric doesn't provide checksums via meta API
}

func (p *FabricProvider) getLatestLoader(ctx context.Context) (string, error) {
	if p.loaderVersion != "" {
		return p.loaderVersion, nil
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, fabricMeta+"/versions/loader", nil)
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var data []struct {
		Version string `json:"version"`
		Stable  bool   `json:"stable"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	for _, v := range data {
		if v.Stable {
			return v.Version, nil
		}
	}
	return "", fmt.Errorf("no stable fabric loader found")
}
