package github

import (
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"runtime"
	"strings"
)

// Release represents a GitHub release
type Release struct {
	TagName string  `json:"tag_name"`
	Name    string  `json:"name"`
	Assets  []Asset `json:"assets"`
}

// Asset represents a release asset
type Asset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// Client handles GitHub API interactions
type Client struct {
	BaseURL string
}

// NewClient creates a new GitHub client
func NewClient() *Client {
	return &Client{
		BaseURL: "https://api.github.com",
	}
}

// GetLatestRelease gets the latest release for a repository
func (c *Client) GetLatestRelease(owner, repo string) (*Release, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/latest", c.BaseURL, owner, repo)
	
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API error: %d", resp.StatusCode)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &release, nil
}

// GetRelease gets a specific release by tag
func (c *Client) GetRelease(owner, repo, tag string) (*Release, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/releases/tags/%s", c.BaseURL, owner, repo, tag)
	
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API error: %d", resp.StatusCode)
	}

	var release Release
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &release, nil
}

// FindAssetForPlatform finds the best asset for current platform
func (r *Release) FindAssetForPlatform(platform string) (*Asset, error) {
	if platform == "" {
		platform = fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
	}

	// Platform mappings
	platformMap := map[string][]string{
		"windows-amd64": {"windows", "win64", "amd64", "x86_64"},
		"windows-386":   {"windows", "win32", "386", "i386"},
		"darwin-amd64":  {"darwin", "macos", "osx", "amd64", "x86_64"},
		"darwin-arm64":  {"darwin", "macos", "osx", "arm64", "aarch64"},
		"linux-amd64":   {"linux", "amd64", "x86_64"},
		"linux-386":     {"linux", "386", "i386"},
		"linux-arm64":   {"linux", "arm64", "aarch64"},
		"linux-arm":     {"linux", "arm", "armv7"},
	}

	keywords := platformMap[platform]
	if len(keywords) == 0 {
		return nil, fmt.Errorf("unsupported platform: %s", platform)
	}

	// Score assets based on platform keywords
	var bestAsset *Asset
	bestScore := 0

	for _, asset := range r.Assets {
		score := r.scorePlatformMatch(asset.Name, keywords)
		if score > bestScore {
			bestScore = score
			bestAsset = &asset
		}
	}

	if bestAsset == nil {
		return nil, fmt.Errorf("no asset found for platform: %s", platform)
	}

	return bestAsset, nil
}

// scorePlatformMatch scores how well an asset name matches platform keywords
func (r *Release) scorePlatformMatch(assetName string, keywords []string) int {
	name := strings.ToLower(assetName)
	score := 0

	for _, keyword := range keywords {
		if strings.Contains(name, strings.ToLower(keyword)) {
			score++
		}
	}

	// Bonus for common archive formats
	if strings.HasSuffix(name, ".zip") || strings.HasSuffix(name, ".tar.gz") {
		score++
	}

	// Penalty for source code archives
	if strings.Contains(name, "source") || strings.Contains(name, "src") {
		score -= 10
	}

	return score
}

// FindSignatureAsset finds signature file for an asset
func (r *Release) FindSignatureAsset(assetName string) (*Asset, error) {
	baseName := strings.TrimSuffix(assetName, filepath.Ext(assetName))
	
	// Common signature file patterns
	patterns := []string{
		assetName + ".sha256",
		assetName + ".sha256sum",
		assetName + ".sig",
		baseName + ".sha256",
		baseName + ".sha256sum",
		"checksums.txt",
		"CHECKSUMS",
		"SHA256SUMS",
	}

	for _, pattern := range patterns {
		for _, asset := range r.Assets {
			if strings.EqualFold(asset.Name, pattern) {
				return &asset, nil
			}
		}
	}

	return nil, fmt.Errorf("no signature found for asset: %s", assetName)
}

// ParseRepoURL parses GitHub repository URL or identifier
func ParseRepoURL(input string) (owner, repo string, err error) {
	// Handle "github:owner/repo" format
	if strings.HasPrefix(input, "github:") {
		input = strings.TrimPrefix(input, "github:")
	}

	// Handle full URLs
	if strings.Contains(input, "github.com/") {
		parts := strings.Split(input, "github.com/")
		if len(parts) != 2 {
			return "", "", fmt.Errorf("invalid GitHub URL: %s", input)
		}
		input = parts[1]
	}

	// Parse owner/repo
	parts := strings.Split(strings.Trim(input, "/"), "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid repository format: %s (expected: owner/repo)", input)
	}

	return parts[0], parts[1], nil
}