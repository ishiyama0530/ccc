package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const latestReleaseAPIURL = "https://api.github.com/repos/ishiyama0530/ccc/releases/latest"

type GitHubReleaseUpdateNotifier struct {
	Client *http.Client
	APIURL string
}

type latestReleaseResponse struct {
	TagName string `json:"tag_name"`
}

func (notifier GitHubReleaseUpdateNotifier) Notice(ctx context.Context, currentVersion string) (string, error) {
	version := strings.TrimSpace(currentVersion)
	if version == "" || version == "dev" {
		return "", nil
	}

	client := notifier.Client
	if client == nil {
		client = &http.Client{Timeout: 1500 * time.Millisecond}
	}

	apiURL := notifier.APIURL
	if apiURL == "" {
		apiURL = latestReleaseAPIURL
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("update check failed: %s", resp.Status)
	}

	var payload latestReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}

	latestVersion := strings.TrimSpace(payload.TagName)
	if latestVersion == "" {
		return "", errors.New("update check failed: empty tag")
	}
	if latestVersion == version {
		return "", nil
	}

	return fmt.Sprintf("アップデート可能です: %s -> %s", version, latestVersion), nil
}
