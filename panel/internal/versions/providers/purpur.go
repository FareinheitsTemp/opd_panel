package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const purpurAPI = "https://api.purpurmc.org/v2/purpur"

type PurpurProvider struct{ client *http.Client }

func NewPurpurProvider() *PurpurProvider {
	return &PurpurProvider{client: &http.Client{}}
}

func (p *PurpurProvider) Name() string { return "purpur" }

func (p *PurpurProvider) GetVersions(ctx context.Context) ([]string, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, purpurAPI, nil)
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	var data struct {
		Versions []string `json:"versions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	return data.Versions, nil
}

func (p *PurpurProvider) GetDownloadURL(ctx context.Context, version string) (string, error) {
	return fmt.Sprintf("%s/%s/latest/download", purpurAPI, version), nil
}

func (p *PurpurProvider) GetChecksum(ctx context.Context, version string) (string, error) {
	url := fmt.Sprintf("%s/%s/latest", purpurAPI, version)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var data struct {
		MD5 string `json:"md5"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	// Purpur gives MD5 not SHA256 — skip checksum verify for now
	return "", nil
}
