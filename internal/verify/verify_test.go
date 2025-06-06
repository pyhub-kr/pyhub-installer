package verify

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestNewVerifier(t *testing.T) {
	v := NewVerifier("/path/to/file")
	
	if v.FilePath != "/path/to/file" {
		t.Errorf("Expected FilePath to be /path/to/file, got %s", v.FilePath)
	}
	
	if v.SignatureType != "" {
		t.Errorf("Expected SignatureType to be empty, got %s", v.SignatureType)
	}
}

func TestDetectSignatureType(t *testing.T) {
	v := &Verifier{}
	
	tests := []struct {
		name      string
		signature string
		wantType  string
	}{
		{
			name:      "SHA256 hash only",
			signature: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
			wantType:  "sha256",
		},
		{
			name:      "SHA256 with filename",
			signature: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  file.txt",
			wantType:  "sha256",
		},
		{
			name:      "SHA256 with spaces",
			signature: "  e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  ",
			wantType:  "sha256",
		},
		{
			name:      "SHA512 hash",
			signature: "cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e",
			wantType:  "sha512",
		},
		{
			name:      "GPG signature",
			signature: "-----BEGIN PGP SIGNATURE-----\nsome signature data\n-----END PGP SIGNATURE-----",
			wantType:  "gpg",
		},
		{
			name:      "Unknown format",
			signature: "abc123",
			wantType:  "unknown",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sigType := v.detectSignatureType(tt.signature)
			if sigType != tt.wantType {
				t.Errorf("Expected signature type %s, got %s", tt.wantType, sigType)
			}
		})
	}
}

func TestGetSHA256(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "verify_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create test file
	testFile := filepath.Join(tempDir, "test.txt")
	content := []byte("Hello, World!")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}
	
	// Calculate expected hash
	h := sha256.New()
	h.Write(content)
	expectedHash := hex.EncodeToString(h.Sum(nil))
	
	// Test GetSHA256
	v := NewVerifier(testFile)
	actualHash, err := v.GetSHA256()
	if err != nil {
		t.Fatalf("GetSHA256 failed: %v", err)
	}
	
	if actualHash != expectedHash {
		t.Errorf("Expected hash %s, got %s", expectedHash, actualHash)
	}
}

func TestVerifySHA256(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "verify_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create test file
	testFile := filepath.Join(tempDir, "test.txt")
	content := []byte("Test content for SHA256 verification")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}
	
	// Calculate hash
	h := sha256.New()
	h.Write(content)
	correctHash := hex.EncodeToString(h.Sum(nil))
	
	v := NewVerifier(testFile)
	
	tests := []struct {
		name        string
		hash        string
		wantErr     bool
	}{
		{
			name:    "Correct hash",
			hash:    correctHash,
			wantErr: false,
		},
		{
			name:    "Correct hash with filename",
			hash:    correctHash + "  test.txt",
			wantErr: false,
		},
		{
			name:    "Correct hash uppercase",
			hash:    hexToUpper(correctHash),
			wantErr: false,
		},
		{
			name:    "Incorrect hash",
			hash:    "0000000000000000000000000000000000000000000000000000000000000000",
			wantErr: true,
		},
		{
			name:    "Invalid hash format",
			hash:    "invalid",
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := v.verifySHA256(tt.hash)
			if (err != nil) != tt.wantErr {
				t.Errorf("verifySHA256() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestVerifyWithString(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "verify_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create test file
	testFile := filepath.Join(tempDir, "test.bin")
	content := []byte("Binary test content")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}
	
	// Calculate hash
	h := sha256.New()
	h.Write(content)
	hash := hex.EncodeToString(h.Sum(nil))
	
	v := NewVerifier(testFile)
	
	// Test successful verification
	err = v.VerifyWithString(hash)
	if err != nil {
		t.Errorf("VerifyWithString failed: %v", err)
	}
	
	if v.SignatureType != "sha256" {
		t.Errorf("Expected SignatureType to be sha256, got %s", v.SignatureType)
	}
	
	// Test failed verification
	err = v.VerifyWithString("invalid_hash")
	if err == nil {
		t.Error("Expected error for invalid hash, got nil")
	}
}

func TestVerifyWithURL(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "verify_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create test file
	testFile := filepath.Join(tempDir, "download.tar.gz")
	content := []byte("Downloaded file content")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatal(err)
	}
	
	// Calculate hash
	h := sha256.New()
	h.Write(content)
	hash := hex.EncodeToString(h.Sum(nil))
	
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/valid.sha256" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(hash + "  download.tar.gz\n"))
		} else if r.URL.Path == "/invalid.sha256" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("0000000000000000000000000000000000000000000000000000000000000000"))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	
	v := NewVerifier(testFile)
	
	// Test successful verification
	err = v.VerifyWithURL(server.URL + "/valid.sha256")
	if err != nil {
		t.Errorf("VerifyWithURL failed: %v", err)
	}
	
	// Test failed verification
	err = v.VerifyWithURL(server.URL + "/invalid.sha256")
	if err == nil {
		t.Error("Expected error for invalid signature, got nil")
	}
	
	// Test 404 response
	err = v.VerifyWithURL(server.URL + "/notfound.sha256")
	if err == nil {
		t.Error("Expected error for 404 response, got nil")
	}
}

func TestDownloadSignature(t *testing.T) {
	// Create test server
	expectedSig := "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/signature.txt" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(expectedSig + "\n"))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	
	v := &Verifier{}
	
	// Test successful download
	sig, err := v.downloadSignature(server.URL + "/signature.txt")
	if err != nil {
		t.Fatalf("downloadSignature failed: %v", err)
	}
	
	if sig != expectedSig {
		t.Errorf("Expected signature %s, got %s", expectedSig, sig)
	}
	
	// Test 404
	_, err = v.downloadSignature(server.URL + "/notfound.txt")
	if err == nil {
		t.Error("Expected error for 404, got nil")
	}
}

func TestVerifySHA512(t *testing.T) {
	v := &Verifier{}
	
	// Currently not implemented
	err := v.verifySHA512("any_hash")
	if err == nil {
		t.Error("Expected error for unimplemented SHA512, got nil")
	}
}

func TestFileNotFound(t *testing.T) {
	v := NewVerifier("/nonexistent/file.txt")
	
	// Test GetSHA256 with non-existent file
	_, err := v.GetSHA256()
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
	
	// Test verifySHA256 with non-existent file
	err = v.verifySHA256("any_hash")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

// Helper function to convert hex string to uppercase
func hexToUpper(s string) string {
	result := ""
	for _, c := range s {
		if c >= 'a' && c <= 'f' {
			result += string(c - 32)
		} else {
			result += string(c)
		}
	}
	return result
}