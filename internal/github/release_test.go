package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
)

func TestNewClient(t *testing.T) {
	client := NewClient()
	
	if client.BaseURL != "https://api.github.com" {
		t.Errorf("Expected BaseURL to be https://api.github.com, got %s", client.BaseURL)
	}
}

func TestParseRepoURL(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantOwner   string
		wantRepo    string
		wantErr     bool
	}{
		{
			name:      "Simple owner/repo",
			input:     "owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "GitHub prefix",
			input:     "github:owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "Full GitHub URL",
			input:     "https://github.com/owner/repo",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "GitHub URL with trailing slash",
			input:     "https://github.com/owner/repo/",
			wantOwner: "owner",
			wantRepo:  "repo",
			wantErr:   false,
		},
		{
			name:      "Complex GitHub URL",
			input:     "https://github.com/pyhub-kr/pyhub-installer",
			wantOwner: "pyhub-kr",
			wantRepo:  "pyhub-installer",
			wantErr:   false,
		},
		{
			name:    "Invalid format - no slash",
			input:   "invalidformat",
			wantErr: true,
		},
		{
			name:    "Invalid format - too many parts",
			input:   "owner/repo/extra",
			wantErr: true,
		},
		{
			name:    "Empty input",
			input:   "",
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := ParseRepoURL(tt.input)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseRepoURL() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr {
				if owner != tt.wantOwner {
					t.Errorf("Expected owner %s, got %s", tt.wantOwner, owner)
				}
				if repo != tt.wantRepo {
					t.Errorf("Expected repo %s, got %s", tt.wantRepo, repo)
				}
			}
		})
	}
}

func TestGetLatestRelease(t *testing.T) {
	// Create test server
	release := Release{
		TagName: "v1.0.0",
		Name:    "Release 1.0.0",
		Assets: []Asset{
			{
				Name:               "app-linux-amd64.tar.gz",
				BrowserDownloadURL: "https://github.com/owner/repo/releases/download/v1.0.0/app-linux-amd64.tar.gz",
				Size:               1024000,
			},
		},
	}
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/releases/latest" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(release)
	}))
	defer server.Close()
	
	client := &Client{BaseURL: server.URL}
	
	// Test successful request
	got, err := client.GetLatestRelease("owner", "repo")
	if err != nil {
		t.Fatalf("GetLatestRelease() error = %v", err)
	}
	
	if got.TagName != release.TagName {
		t.Errorf("Expected TagName %s, got %s", release.TagName, got.TagName)
	}
	
	if len(got.Assets) != len(release.Assets) {
		t.Errorf("Expected %d assets, got %d", len(release.Assets), len(got.Assets))
	}
	
	// Test 404 response
	_, err = client.GetLatestRelease("invalid", "repo")
	if err == nil {
		t.Error("Expected error for 404 response, got nil")
	}
}

func TestGetRelease(t *testing.T) {
	// Create test server
	release := Release{
		TagName: "v1.2.3",
		Name:    "Release 1.2.3",
		Assets: []Asset{
			{
				Name:               "app-windows-amd64.zip",
				BrowserDownloadURL: "https://github.com/owner/repo/releases/download/v1.2.3/app-windows-amd64.zip",
				Size:               2048000,
			},
		},
	}
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/owner/repo/releases/tags/v1.2.3" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(release)
	}))
	defer server.Close()
	
	client := &Client{BaseURL: server.URL}
	
	// Test successful request
	got, err := client.GetRelease("owner", "repo", "v1.2.3")
	if err != nil {
		t.Fatalf("GetRelease() error = %v", err)
	}
	
	if got.TagName != release.TagName {
		t.Errorf("Expected TagName %s, got %s", release.TagName, got.TagName)
	}
	
	// Test invalid tag
	_, err = client.GetRelease("owner", "repo", "invalid-tag")
	if err == nil {
		t.Error("Expected error for invalid tag, got nil")
	}
}

func TestFindAssetForPlatform(t *testing.T) {
	release := &Release{
		Assets: []Asset{
			{Name: "app-windows-amd64.exe", Size: 1000},
			{Name: "app-windows-386.exe", Size: 900},
			{Name: "app-darwin-amd64.tar.gz", Size: 1100},
			{Name: "app-darwin-arm64.tar.gz", Size: 1050},
			{Name: "app-linux-amd64.tar.gz", Size: 1200},
			{Name: "app-linux-arm64.tar.gz", Size: 1150},
			{Name: "app-source.tar.gz", Size: 500},
		},
	}
	
	tests := []struct {
		name         string
		platform     string
		wantAsset    string
		wantErr      bool
	}{
		{
			name:      "Windows AMD64",
			platform:  "windows-amd64",
			wantAsset: "app-windows-amd64.exe",
			wantErr:   false,
		},
		{
			name:      "Darwin ARM64",
			platform:  "darwin-arm64",
			wantAsset: "app-darwin-arm64.tar.gz",
			wantErr:   false,
		},
		{
			name:      "Linux AMD64",
			platform:  "linux-amd64",
			wantAsset: "app-linux-amd64.tar.gz",
			wantErr:   false,
		},
		{
			name:      "Auto-detect platform",
			platform:  "",
			wantAsset: "", // Will depend on runtime
			wantErr:   false,
		},
		{
			name:     "Unsupported platform",
			platform: "freebsd-amd64",
			wantErr:  true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asset, err := release.FindAssetForPlatform(tt.platform)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("FindAssetForPlatform() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && tt.platform != "" {
				if asset.Name != tt.wantAsset {
					t.Errorf("Expected asset %s, got %s", tt.wantAsset, asset.Name)
				}
			}
			
			// For auto-detect, just ensure we got a valid asset
			if !tt.wantErr && tt.platform == "" && asset == nil {
				t.Error("Expected non-nil asset for auto-detect")
			}
		})
	}
}

func TestScorePlatformMatch(t *testing.T) {
	release := &Release{}
	
	tests := []struct {
		name      string
		assetName string
		keywords  []string
		wantScore int
	}{
		{
			name:      "Perfect match with archive",
			assetName: "app-linux-amd64.tar.gz",
			keywords:  []string{"linux", "amd64"},
			wantScore: 3, // 2 keywords + 1 for .tar.gz
		},
		{
			name:      "Match with source penalty",
			assetName: "app-source-linux.tar.gz",
			keywords:  []string{"linux"},
			wantScore: -8, // 1 keyword + 1 for .tar.gz - 10 for source
		},
		{
			name:      "No match",
			assetName: "app-windows.exe",
			keywords:  []string{"linux", "amd64"},
			wantScore: 0,
		},
		{
			name:      "Partial match",
			assetName: "app-darwin-universal.zip",
			keywords:  []string{"darwin", "amd64"},
			wantScore: 2, // 1 keyword + 1 for .zip
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := release.scorePlatformMatch(tt.assetName, tt.keywords)
			if score != tt.wantScore {
				t.Errorf("Expected score %d, got %d", tt.wantScore, score)
			}
		})
	}
}

func TestFindSignatureAsset(t *testing.T) {
	release := &Release{
		Assets: []Asset{
			{Name: "app-linux-amd64.tar.gz"},
			{Name: "app-linux-amd64.tar.gz.sha256"},
			{Name: "app-windows.exe"},
			{Name: "app-windows.exe.sig"},
		},
	}
	
	tests := []struct {
		name          string
		assetName     string
		wantSignature string
		wantErr       bool
	}{
		{
			name:          "Find .sha256 signature",
			assetName:     "app-linux-amd64.tar.gz",
			wantSignature: "app-linux-amd64.tar.gz.sha256",
			wantErr:       false,
		},
		{
			name:          "Find .sig signature",
			assetName:     "app-windows.exe",
			wantSignature: "app-windows.exe.sig",
			wantErr:       false,
		},
		{
			name:      "No signature found",
			assetName: "app-darwin.dmg",
			wantErr:   true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asset, err := release.FindSignatureAsset(tt.assetName)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("FindSignatureAsset() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && asset.Name != tt.wantSignature {
				t.Errorf("Expected signature %s, got %s", tt.wantSignature, asset.Name)
			}
		})
	}
}

func TestFindSignatureAssetWithGenericChecksums(t *testing.T) {
	release := &Release{
		Assets: []Asset{
			{Name: "app-linux-amd64.tar.gz"},
			{Name: "app-windows.exe"},
			{Name: "app-darwin.dmg"},
			{Name: "SHA256SUMS"},
			{Name: "checksums.txt"},
		},
	}
	
	tests := []struct {
		name          string
		assetName     string
		wantSignature string
		wantErr       bool
	}{
		{
			name:          "Find generic SHA256SUMS",
			assetName:     "app-linux-amd64.tar.gz",
			wantSignature: "checksums.txt",
			wantErr:       false,
		},
		{
			name:          "Find generic checksums for any file",
			assetName:     "app-darwin.dmg",
			wantSignature: "checksums.txt",
			wantErr:       false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			asset, err := release.FindSignatureAsset(tt.assetName)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("FindSignatureAsset() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && asset.Name != tt.wantSignature {
				t.Errorf("Expected signature %s, got %s", tt.wantSignature, asset.Name)
			}
		})
	}
}

func TestFindAssetForCurrentPlatform(t *testing.T) {
	// Test with current runtime platform
	release := &Release{
		Assets: []Asset{
			{Name: "app-windows-amd64.exe"},
			{Name: "app-darwin-amd64.tar.gz"},
			{Name: "app-darwin-arm64.tar.gz"},
			{Name: "app-linux-amd64.tar.gz"},
			{Name: "app-linux-arm64.tar.gz"},
		},
	}
	
	// Use empty platform to trigger auto-detection
	asset, err := release.FindAssetForPlatform("")
	
	if err != nil {
		t.Fatalf("FindAssetForPlatform() with auto-detect failed: %v", err)
	}
	
	// Verify the asset matches current platform
	currentPlatform := runtime.GOOS + "-" + runtime.GOARCH
	if !containsString(asset.Name, runtime.GOOS) {
		t.Errorf("Asset %s doesn't match current OS %s", asset.Name, runtime.GOOS)
	}
	
	t.Logf("Auto-detected platform %s, selected asset: %s", currentPlatform, asset.Name)
}

func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || substringIndex(s, substr) >= 0))
}

func substringIndex(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}