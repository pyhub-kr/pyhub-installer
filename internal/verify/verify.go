package verify

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

// Verifier handles file signature verification
type Verifier struct {
	FilePath      string
	SignatureType string // "sha256", "sha512", "gpg"
}

// NewVerifier creates a new verifier
func NewVerifier(filePath string) *Verifier {
	return &Verifier{
		FilePath: filePath,
	}
}

// VerifyWithURL verifies file against signature from URL
func (v *Verifier) VerifyWithURL(signatureURL string) error {
	// Download signature
	signature, err := v.downloadSignature(signatureURL)
	if err != nil {
		return fmt.Errorf("failed to download signature: %w", err)
	}

	// Auto-detect signature type
	v.SignatureType = v.detectSignatureType(signature)

	// Verify based on type
	switch v.SignatureType {
	case "sha256":
		return v.verifySHA256(signature)
	case "sha512":
		return v.verifySHA512(signature)
	default:
		return fmt.Errorf("unsupported signature type: %s", v.SignatureType)
	}
}

// VerifyWithString verifies file against signature string
func (v *Verifier) VerifyWithString(signature string) error {
	v.SignatureType = v.detectSignatureType(signature)
	
	switch v.SignatureType {
	case "sha256":
		return v.verifySHA256(signature)
	case "sha512":
		return v.verifySHA512(signature)
	default:
		return fmt.Errorf("unsupported signature type: %s", v.SignatureType)
	}
}

// downloadSignature downloads signature from URL
func (v *Verifier) downloadSignature(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download signature: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(body)), nil
}

// detectSignatureType detects signature type from content
func (v *Verifier) detectSignatureType(signature string) string {
	signature = strings.TrimSpace(signature)
	
	// Check for GPG signature first (before stripping fields)
	if strings.Contains(signature, "-----BEGIN PGP") {
		return "gpg"
	}
	
	// Remove any filename info (common in checksum files)
	parts := strings.Fields(signature)
	if len(parts) > 0 {
		signature = parts[0]
	}

	switch len(signature) {
	case 64:
		return "sha256"
	case 128:
		return "sha512"
	default:
		return "unknown"
	}
}

// verifySHA256 verifies SHA256 signature
func (v *Verifier) verifySHA256(expectedHash string) error {
	// Clean expected hash (remove filename if present)
	parts := strings.Fields(expectedHash)
	if len(parts) > 0 {
		expectedHash = parts[0]
	}
	expectedHash = strings.TrimSpace(expectedHash)

	// Calculate file hash
	file, err := os.Open(v.FilePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("failed to calculate hash: %w", err)
	}

	actualHash := hex.EncodeToString(hash.Sum(nil))

	// Compare hashes (case insensitive)
	if !strings.EqualFold(actualHash, expectedHash) {
		return fmt.Errorf("SHA256 verification failed:\nExpected: %s\nActual:   %s", expectedHash, actualHash)
	}

	fmt.Printf("✓ SHA256 verification passed: %s\n", actualHash)
	return nil
}

// verifySHA512 verifies SHA512 signature (similar to SHA256)
func (v *Verifier) verifySHA512(expectedHash string) error {
	// TODO: Implement SHA512 verification
	return fmt.Errorf("SHA512 verification not yet implemented")
}

// GetSHA256 calculates SHA256 hash of file
func (v *Verifier) GetSHA256() (string, error) {
	file, err := os.Open(v.FilePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to calculate hash: %w", err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}