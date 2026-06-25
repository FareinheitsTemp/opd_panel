package versions

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

type Progress struct {
	Downloaded int64
	Total      int64
}

type Downloader struct {
	client   *http.Client
	Progress chan Progress
}

func NewDownloader() *Downloader {
	return &Downloader{
		client:   &http.Client{Timeout: 5 * time.Minute},
		Progress: make(chan Progress, 64),
	}
}

func (d *Downloader) Download(ctx context.Context, url, expectedSHA256 string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download returned HTTP %d", resp.StatusCode)
	}

	tmp, err := os.CreateTemp("", "opd-download-*.jar")
	if err != nil {
		return "", err
	}
	defer tmp.Close()

	hasher := sha256.New()
	writer := io.MultiWriter(tmp, hasher)

	buf := make([]byte, 32*1024)
	var downloaded int64
	total := resp.ContentLength

	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, wErr := writer.Write(buf[:n]); wErr != nil {
				return "", wErr
			}
			downloaded += int64(n)
			select {
			case d.Progress <- Progress{Downloaded: downloaded, Total: total}:
			default:
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return "", readErr
		}
	}

	// SHA256 verification
	if expectedSHA256 != "" {
		actual := hex.EncodeToString(hasher.Sum(nil))
		if actual != expectedSHA256 {
			os.Remove(tmp.Name())
			return "", fmt.Errorf("checksum mismatch: expected %s got %s", expectedSHA256, actual)
		}
	}

	return tmp.Name(), nil
}
