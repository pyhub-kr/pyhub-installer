package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/pyhub-kr/pyhub-installer/internal/download"
	"github.com/pyhub-kr/pyhub-installer/internal/verify"
	"github.com/pyhub-kr/pyhub-installer/internal/extract"
	"github.com/pyhub-kr/pyhub-installer/internal/install"
	"github.com/pyhub-kr/pyhub-installer/internal/github"
)

var rootCmd = &cobra.Command{
	Use:   "pyhub-installer",
	Short: "Cross-platform installer for downloading, verifying and installing files",
	Long: `A fast, secure cross-platform installer that:
- Downloads files from URLs with parallel processing
- Verifies signatures (SHA256, auto-detect from GitHub releases)
- Extracts ZIP/TAR archives
- Installs to specified paths with proper permissions
- Supports Windows, macOS, and Linux`,
}

var downloadCmd = &cobra.Command{
	Use:   "download [URL]",
	Short: "Download and install a file from URL",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runDownload(cmd, args); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

var installCmd = &cobra.Command{
	Use:   "install [GITHUB_REPO]",
	Short: "Install from GitHub release (e.g., github:pyhub-kr/pyhub-mcptools)",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runInstall(cmd, args); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	// Download command flags
	downloadCmd.Flags().StringP("output", "o", ".", "Output directory")
	downloadCmd.Flags().BoolP("verify", "v", false, "Verify signature")
	downloadCmd.Flags().BoolP("extract", "x", false, "Extract archive")
	downloadCmd.Flags().StringP("signature", "s", "", "Signature URL for verification")
	downloadCmd.Flags().String("chmod", "755", "File permissions (Unix)")
	downloadCmd.Flags().BoolP("remove-archive", "r", false, "Remove archive after extraction")
	downloadCmd.Flags().BoolP("flatten", "f", false, "Remove top-level directory when extracting")
	downloadCmd.Flags().Bool("no-flatten", false, "Disable automatic flattening of single top-level directory")
	
	// Install command flags
	installCmd.Flags().String("version", "latest", "Version to install")
	installCmd.Flags().String("platform", "", "Target platform (auto-detect if empty)")
	installCmd.Flags().StringP("output", "o", "/usr/local/bin", "Installation directory")
	
	rootCmd.AddCommand(downloadCmd)
	rootCmd.AddCommand(installCmd)
}

// runDownload implements the download command
func runDownload(cmd *cobra.Command, args []string) error {
	url := args[0]
	output, _ := cmd.Flags().GetString("output")
	verifyFlag, _ := cmd.Flags().GetBool("verify")
	extractFlag, _ := cmd.Flags().GetBool("extract")
	signature, _ := cmd.Flags().GetString("signature")
	chmod, _ := cmd.Flags().GetString("chmod")
	removeArchive, _ := cmd.Flags().GetBool("remove-archive")
	flatten, _ := cmd.Flags().GetBool("flatten")
	noFlatten, _ := cmd.Flags().GetBool("no-flatten")

	// If user specified a system directory and doesn't have write permission, find alternative
	systemDirs := []string{"/usr/local/bin", "/usr/bin", "/opt", "/usr/local"}
	isSystemDir := false
	for _, sysDir := range systemDirs {
		if strings.HasPrefix(output, sysDir) {
			isSystemDir = true
			break
		}
	}
	
	if isSystemDir {
		// Try to create directory first to test permission
		if err := os.MkdirAll(output, 0755); err != nil {
			if writableDir, pathErr := install.FindWritableInstallPath(); pathErr == nil {
				fmt.Printf("Permission denied for %s, using writable directory: %s\n", output, writableDir)
				output = writableDir
			}
		}
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(output, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Determine filename from URL
	filename := filepath.Base(url)
	if filename == "/" || filename == "." {
		filename = "download"
	}
	
	// Create full output path
	outputPath := filepath.Join(output, filename)

	fmt.Printf("Downloading %s...\n", url)

	// Download file
	downloader := download.NewChunkDownloader(url, outputPath)
	ctx := context.Background()
	if err := downloader.Download(ctx); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	fmt.Printf("✓ Downloaded to: %s\n", outputPath)

	// Verify signature if requested
	if verifyFlag && signature != "" {
		fmt.Println("Verifying signature...")
		verifier := verify.NewVerifier(outputPath)
		if err := verifier.VerifyWithURL(signature); err != nil {
			return fmt.Errorf("verification failed: %w", err)
		}
	}

	// Extract if requested
	if extractFlag {
		fmt.Println("Extracting archive...")
		extractor := extract.NewExtractor(outputPath, output)
		
		// Configure flatten behavior
		if flatten {
			extractor.SetFlatten(true)
		} else if !noFlatten {
			// Auto-detect single top-level directory by default
			extractor.SetAutoFlatten(true)
		}
		
		if err := extractor.Extract(); err != nil {
			return fmt.Errorf("extraction failed: %w", err)
		}
		
		// Remove archive after successful extraction if requested
		if removeArchive {
			fmt.Printf("Removing archive: %s\n", outputPath)
			if err := os.Remove(outputPath); err != nil {
				fmt.Printf("Warning: failed to remove archive: %v\n", err)
			}
		}
	}

	// Install with permissions
	if chmod != "" && !extractFlag {
		installer := install.NewInstaller(outputPath, outputPath, chmod)
		if err := installer.Install(); err != nil {
			return fmt.Errorf("permission setting failed: %w", err)
		}
	}

	return nil
}

// runInstall implements the install command
func runInstall(cmd *cobra.Command, args []string) error {
	repo := args[0]
	version, _ := cmd.Flags().GetString("version")
	platform, _ := cmd.Flags().GetString("platform")
	output, _ := cmd.Flags().GetString("output")

	// If using default output path, try to find a writable directory in PATH
	if output == "/usr/local/bin" {
		if writableDir, err := install.FindWritableInstallPath(); err == nil {
			if writableDir != output {
				fmt.Printf("Using writable directory: %s\n", writableDir)
				output = writableDir
			}
		}
	}

	// Create output directory if it doesn't exist
	if err := os.MkdirAll(output, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Parse repository
	owner, repoName, err := github.ParseRepoURL(repo)
	if err != nil {
		return fmt.Errorf("invalid repository: %w", err)
	}

	fmt.Printf("Installing %s/%s from GitHub...\n", owner, repoName)

	// Get release
	client := github.NewClient()
	var release *github.Release
	
	if version == "latest" {
		release, err = client.GetLatestRelease(owner, repoName)
	} else {
		release, err = client.GetRelease(owner, repoName, version)
	}
	
	if err != nil {
		return fmt.Errorf("failed to get release: %w", err)
	}

	fmt.Printf("Found release: %s\n", release.TagName)

	// Find asset for platform
	asset, err := release.FindAssetForPlatform(platform)
	if err != nil {
		return fmt.Errorf("failed to find asset: %w", err)
	}

	fmt.Printf("Found asset: %s (%d bytes)\n", asset.Name, asset.Size)

	// Download asset
	outputPath := filepath.Join(output, asset.Name)
	downloader := download.NewChunkDownloader(asset.BrowserDownloadURL, outputPath)
	ctx := context.Background()
	
	if err := downloader.Download(ctx); err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// Try to find and verify signature
	sigAsset, err := release.FindSignatureAsset(asset.Name)
	if err == nil {
		fmt.Println("Found signature file, verifying...")
		verifier := verify.NewVerifier(outputPath)
		if err := verifier.VerifyWithURL(sigAsset.BrowserDownloadURL); err != nil {
			fmt.Printf("Warning: signature verification failed: %v\n", err)
		}
	} else {
		fmt.Println("No signature file found, skipping verification")
	}

	// Extract if it's an archive
	extractor := extract.NewExtractor(outputPath, output)
	if err := extractor.Extract(); err != nil {
		fmt.Printf("Note: Not an archive or extraction failed: %v\n", err)
	} else {
		// Set executable permissions for extracted files
		installer := install.NewInstaller(output, output, "755")
		if err := installer.InstallDirectory(); err != nil {
			fmt.Printf("Warning: failed to set permissions: %v\n", err)
		}
	}

	fmt.Printf("✓ Installation completed to: %s\n", output)
	return nil
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}