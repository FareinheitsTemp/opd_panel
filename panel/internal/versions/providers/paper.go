package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

const paperAPI = "https://api.papermc.io/v2/projects/paper"

type PaperProvider struct{ client *http.Client }

func NewPaperProvider() *PaperProvider {
	return &PaperProvider{client: &http.Client{}}
}

func (p *PaperProvider) Name() string { return "paper" }

func (p *PaperProvider) GetVersions(ctx context.Context) ([]string, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, paperAPI, nil)
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
	// Reverse so newest is first
	for i, j := 0, len(data.Versions)-1; i < j; i, j = i+1, j-1 {
		data.Versions[i], data.Versions[j] = data.Versions[j], data.Versions[i]
	}
	return data.Versions, nil
}

func (p *PaperProvider) GetDownloadURL(ctx context.Context, version string) (string, error) {
	buildNum, err := p.getLatestBuild(ctx, version)
	if err != nil {
		return "", err
	}
	url := fmt.Sprintf("%s/versions/%s/builds/%d/downloads/paper-%s-%d.jar",
		paperAPI, version, buildNum, version, buildNum)
	return url, nil
}

func (p *PaperProvider) GetChecksum(ctx context.Context, version string) (string, error) {
	buildNum, err := p.getLatestBuild(ctx, version)
	if err != nil {
		return "", err
	}
	url := fmt.Sprintf("%s/versions/%s/builds/%d", paperAPI, version, buildNum)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := p.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	var data struct {
		Downloads struct {
			Application struct {
				SHA256 string `json:"sha256"`
			} `json:"application"`
		} `json:"downloads"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}
	return data.Downloads.Application.SHA256, nil
}

func (p *PaperProvider) getLatestBuild(ctx context.Context, version string) (int, error) {
	url := fmt.Sprintf("%s/versions/%s/builds", paperAPI, version)
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	resp, err := p.client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	var data struct {
		Builds []struct {
			Build int `json:"build"`
		} `json:"builds"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return 0, err
	}
	if len(data.Builds) == 0 {
		return 0, fmt.Errorf("no builds for paper %s", version)
	}
	return data.Builds[len(data.Builds)-1].Build, nil
}
