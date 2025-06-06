package install

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// Installer handles file installation and permissions
type Installer struct {
	SourcePath string
	DestPath   string
	Chmod      string
}

// NewInstaller creates a new installer
func NewInstaller(sourcePath, destPath, chmod string) *Installer {
	return &Installer{
		SourcePath: sourcePath,
		DestPath:   destPath,
		Chmod:      chmod,
	}
}

// Install installs file to destination with proper permissions
func (i *Installer) Install() error {
	// Ensure destination directory exists
	destDir := filepath.Dir(i.DestPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Copy or move file
	if err := i.copyFile(); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	// Set permissions (Unix only)
	if runtime.GOOS != "windows" {
		if err := i.setPermissions(); err != nil {
			return fmt.Errorf("failed to set permissions: %w", err)
		}
	}

	fmt.Printf("✓ Installed to: %s\n", i.DestPath)
	return nil
}

// InstallDirectory installs all files from source directory
func (i *Installer) InstallDirectory() error {
	return filepath.Walk(i.SourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path
		relPath, err := filepath.Rel(i.SourcePath, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(i.DestPath, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		// Install individual file
		installer := NewInstaller(path, destPath, i.Chmod)
		return installer.Install()
	})
}

// copyFile copies file from source to destination
func (i *Installer) copyFile() error {
	source, err := os.Open(i.SourcePath)
	if err != nil {
		return err
	}
	defer source.Close()

	dest, err := os.Create(i.DestPath)
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = dest.ReadFrom(source)
	return err
}

// setPermissions sets file permissions (Unix only)
func (i *Installer) setPermissions() error {
	if i.Chmod == "" {
		return nil
	}

	// Parse chmod string (e.g., "755", "644")
	mode, err := i.parseChmod(i.Chmod)
	if err != nil {
		return fmt.Errorf("invalid chmod value: %s", i.Chmod)
	}

	return os.Chmod(i.DestPath, mode)
}

// parseChmod parses chmod string to os.FileMode
func (i *Installer) parseChmod(chmod string) (os.FileMode, error) {
	// Handle octal string (e.g., "755")
	if len(chmod) == 3 {
		mode, err := strconv.ParseInt(chmod, 8, 32)
		if err != nil {
			return 0, err
		}
		return os.FileMode(mode), nil
	}

	// Handle symbolic permissions (e.g., "rwxr-xr-x")
	if len(chmod) == 9 {
		return i.parseSymbolicMode(chmod)
	}

	return 0, fmt.Errorf("unsupported chmod format: %s", chmod)
}

// parseSymbolicMode parses symbolic mode string
func (i *Installer) parseSymbolicMode(mode string) (os.FileMode, error) {
	var perm os.FileMode

	// Owner permissions
	if mode[0] == 'r' {
		perm |= 0400
	}
	if mode[1] == 'w' {
		perm |= 0200
	}
	if mode[2] == 'x' {
		perm |= 0100
	}

	// Group permissions
	if mode[3] == 'r' {
		perm |= 0040
	}
	if mode[4] == 'w' {
		perm |= 0020
	}
	if mode[5] == 'x' {
		perm |= 0010
	}

	// Other permissions
	if mode[6] == 'r' {
		perm |= 0004
	}
	if mode[7] == 'w' {
		perm |= 0002
	}
	if mode[8] == 'x' {
		perm |= 0001
	}

	return perm, nil
}

// FindExecutables finds executable files in a directory
func FindExecutables(dirPath string) ([]string, error) {
	var executables []string
	installer := &Installer{} // Create instance for method access

	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		// Check if file is executable
		if installer.isExecutable(path, info) {
			executables = append(executables, path)
		}

		return nil
	})

	return executables, err
}

// isExecutable checks if file is executable
func (i *Installer) isExecutable(path string, info os.FileInfo) bool {
	// Windows: check file extension
	if runtime.GOOS == "windows" {
		ext := strings.ToLower(filepath.Ext(path))
		return ext == ".exe" || ext == ".bat" || ext == ".cmd" || ext == ".ps1"
	}

	// Unix: check permissions
	return info.Mode()&0111 != 0
}

// AddToPath adds directory to system PATH (platform-specific)
func AddToPath(dirPath string) error {
	installer := &Installer{} // Create instance for method access
	switch runtime.GOOS {
	case "windows":
		return installer.addToPathWindows(dirPath)
	case "darwin", "linux":
		return installer.addToPathUnix(dirPath)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// addToPathWindows adds to PATH on Windows
func (i *Installer) addToPathWindows(dirPath string) error {
	// TODO: Implement Windows PATH modification
	fmt.Printf("Note: Add %s to your PATH manually on Windows\n", dirPath)
	return nil
}

// addToPathUnix adds to PATH on Unix systems
func (i *Installer) addToPathUnix(dirPath string) error {
	// TODO: Implement Unix PATH modification
	fmt.Printf("Note: Add %s to your PATH manually:\n", dirPath)
	fmt.Printf("  export PATH=\"%s:$PATH\"\n", dirPath)
	return nil
}