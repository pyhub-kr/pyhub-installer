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

// GetStandardInstallPath returns the standard installation path for a program
// Returns installPath (where files are installed) and binPath (where executables/links go)
func GetStandardInstallPath(programName string) (installPath string, binPath string, err error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	
	switch runtime.GOOS {
	case "windows":
		// Windows: Each program gets its own directory, add to PATH
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			localAppData = filepath.Join(homeDir, "AppData", "Local")
		}
		installPath = filepath.Join(localAppData, "Programs", programName)
		binPath = installPath // Windows adds program directory to PATH
		
	case "darwin", "linux":
		// Unix-like: Install to share, symlink to bin
		installPath = filepath.Join(homeDir, ".local", "share", programName)
		binPath = filepath.Join(homeDir, ".local", "bin")
		
	default:
		// Fallback for other systems
		installPath = filepath.Join(homeDir, ".local", "share", programName)
		binPath = filepath.Join(homeDir, ".local", "bin")
	}
	
	return installPath, binPath, nil
}

// IsPathInEnv checks if a directory is in the PATH environment variable
func IsPathInEnv(dirPath string) bool {
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return false
	}
	
	separator := ":"
	if runtime.GOOS == "windows" {
		separator = ";"
	}
	
	paths := strings.Split(pathEnv, separator)
	cleanDir := filepath.Clean(dirPath)
	
	for _, p := range paths {
		if filepath.Clean(p) == cleanDir {
			return true
		}
	}
	
	return false
}

// InstallStrategy defines how to install a program
type InstallStrategy interface {
	Install(source string, programName string) error
}

// DirectoryInstallStrategy installs a directory with multiple files
type DirectoryInstallStrategy struct {
	InstallPath string
	BinPath     string
}

// Install installs a directory to the standard location
func (s *DirectoryInstallStrategy) Install(sourceDir string, programName string) error {
	// Create installation directory
	if err := os.MkdirAll(s.InstallPath, 0755); err != nil {
		return fmt.Errorf("failed to create installation directory: %w", err)
	}
	
	// Copy all files from source to install path
	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		
		// Calculate relative path
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		
		destPath := filepath.Join(s.InstallPath, relPath)
		
		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}
		
		// Copy file
		return copyFileWithPermissions(path, destPath, info.Mode())
	})
	
	if err != nil {
		return fmt.Errorf("failed to copy directory: %w", err)
	}
	
	// Find executables and create symlinks (Unix) or add to PATH (Windows)
	if runtime.GOOS != "windows" {
		return s.createSymlinks(programName)
	}
	
	return nil
}

// createSymlinks creates symbolic links for executables in bin directory
func (s *DirectoryInstallStrategy) createSymlinks(programName string) error {
	// Create bin directory if it doesn't exist
	if err := os.MkdirAll(s.BinPath, 0755); err != nil {
		return fmt.Errorf("failed to create bin directory: %w", err)
	}
	
	// Find executable files
	executables, err := FindExecutables(s.InstallPath)
	if err != nil {
		return fmt.Errorf("failed to find executables: %w", err)
	}
	
	// Create symlinks for each executable
	for _, exe := range executables {
		exeName := filepath.Base(exe)
		linkPath := filepath.Join(s.BinPath, exeName)
		
		// Remove existing link if any
		os.Remove(linkPath)
		
		// Create new symlink
		if err := os.Symlink(exe, linkPath); err != nil {
			return fmt.Errorf("failed to create symlink for %s: %w", exeName, err)
		}
		
		fmt.Printf("✓ Created symlink: %s -> %s\n", linkPath, exe)
	}
	
	return nil
}

// SingleFileInstallStrategy installs a single executable file
type SingleFileInstallStrategy struct {
	InstallPath string
	BinPath     string
}

// Install installs a single file to the standard location
func (s *SingleFileInstallStrategy) Install(sourceFile string, programName string) error {
	// Create installation directory
	if err := os.MkdirAll(s.InstallPath, 0755); err != nil {
		return fmt.Errorf("failed to create installation directory: %w", err)
	}
	
	// Determine destination file name
	fileName := filepath.Base(sourceFile)
	destFile := filepath.Join(s.InstallPath, fileName)
	
	// Copy file with permissions
	info, err := os.Stat(sourceFile)
	if err != nil {
		return fmt.Errorf("failed to stat source file: %w", err)
	}
	
	if err := copyFileWithPermissions(sourceFile, destFile, info.Mode()); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}
	
	// Create symlink (Unix) or copy to bin (Windows)
	if runtime.GOOS != "windows" {
		// Create bin directory if needed
		if err := os.MkdirAll(s.BinPath, 0755); err != nil {
			return fmt.Errorf("failed to create bin directory: %w", err)
		}
		
		linkPath := filepath.Join(s.BinPath, fileName)
		os.Remove(linkPath) // Remove existing link if any
		
		if err := os.Symlink(destFile, linkPath); err != nil {
			return fmt.Errorf("failed to create symlink: %w", err)
		}
		
		fmt.Printf("✓ Created symlink: %s -> %s\n", linkPath, destFile)
	}
	
	fmt.Printf("✓ Installed to: %s\n", s.InstallPath)
	return nil
}

// copyFileWithPermissions copies a file preserving permissions
func copyFileWithPermissions(src, dst string, mode os.FileMode) error {
	input, err := os.Open(src)
	if err != nil {
		return err
	}
	defer input.Close()
	
	output, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer output.Close()
	
	if _, err := output.ReadFrom(input); err != nil {
		return err
	}
	
	// Set permissions
	return os.Chmod(dst, mode)
}

// getPathDirectories returns directories from PATH environment variable in priority order
func getPathDirectories() []string {
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return []string{}
	}
	
	separator := ":"
	if runtime.GOOS == "windows" {
		separator = ";"
	}
	
	dirs := strings.Split(pathEnv, separator)
	
	// Filter and prioritize directories into three groups
	var highPriority []string    // User and system tool directories
	var normalPriority []string  // Other directories
	var lowPriority []string     // Language-specific directories
	
	homeDir, _ := os.UserHomeDir()
	
	for _, dir := range dirs {
		if dir == "" {
			continue
		}
		
		// Clean path
		dir = filepath.Clean(dir)
		
		// Skip problematic directories
		if isProblematicPath(dir) {
			continue
		}
		
		// Check if it's a language-specific directory
		if isLanguageSpecificPath(dir) {
			lowPriority = append(lowPriority, dir)
			continue
		}
		
		// Prioritize user-local directories (but not language-specific ones)
		if strings.HasPrefix(dir, homeDir) && !isLanguageSpecificPath(dir) {
			highPriority = append(highPriority, dir)
		} else if isPreferredSystemPath(dir) {
			highPriority = append(highPriority, dir)
		} else {
			normalPriority = append(normalPriority, dir)
		}
	}
	
	// Return in priority order: high, normal, then language-specific as last resort
	result := append(highPriority, normalPriority...)
	return append(result, lowPriority...)
}

// getFallbackDirectories returns fallback directories if no PATH directory is writable
func getFallbackDirectories() []string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return []string{"."}
	}
	
	var fallbacks []string
	
	// Add platform-specific fallbacks
	switch runtime.GOOS {
	case "windows":
		// Windows specific paths
		if localAppData := os.Getenv("LOCALAPPDATA"); localAppData != "" {
			fallbacks = append(fallbacks, filepath.Join(localAppData, "Programs"))
		}
		if programFiles := os.Getenv("ProgramFiles"); programFiles != "" {
			fallbacks = append(fallbacks, filepath.Join(programFiles, "pyhub-installer"))
		}
		fallbacks = append(fallbacks, 
			filepath.Join(homeDir, "bin"),
			filepath.Join(homeDir, ".local", "bin"),
		)
	case "darwin":
		// macOS specific paths
		fallbacks = []string{
			filepath.Join(homeDir, ".local", "bin"),
			filepath.Join(homeDir, "bin"),
			"/opt/homebrew/bin",
			"/usr/local/bin",
		}
	case "linux":
		// Linux specific paths
		fallbacks = []string{
			filepath.Join(homeDir, ".local", "bin"),
			filepath.Join(homeDir, "bin"),
			"/usr/local/bin",
		}
	default:
		fallbacks = []string{
			filepath.Join(homeDir, ".local", "bin"),
			filepath.Join(homeDir, "bin"),
		}
	}
	
	return fallbacks
}

// isDirectoryWritable checks if a directory is writable
func isDirectoryWritable(dir string) bool {
	// Check if directory exists
	info, err := os.Stat(dir)
	if err != nil {
		return false
	}
	
	if !info.IsDir() {
		return false
	}
	
	// Try to create a temporary file to test writability
	testFile := filepath.Join(dir, ".write_test_"+strconv.Itoa(os.Getpid()))
	file, err := os.Create(testFile)
	if err != nil {
		return false
	}
	
	file.Close()
	os.Remove(testFile)
	return true
}

// isProblematicPath checks if a path should be skipped
func isProblematicPath(dir string) bool {
	// Skip empty or current directory
	if dir == "" || dir == "." {
		return true
	}
	
	// Skip network paths (UNC paths on Windows)
	if runtime.GOOS == "windows" && strings.HasPrefix(dir, "\\\\") {
		return true
	}
	
	// Skip some system directories that are typically read-only
	problematicPaths := []string{
		"/sbin",
		"/usr/sbin",
		"/System",
		"/Windows",
		"/Program Files",
	}
	
	for _, problematic := range problematicPaths {
		if strings.HasPrefix(dir, problematic) {
			return true
		}
	}
	
	return false
}

// isPreferredSystemPath checks if a system path is preferred
func isPreferredSystemPath(dir string) bool {
	// Normalize for comparison
	normalizedDir := strings.ToLower(filepath.ToSlash(dir))
	
	// Windows preferred paths
	if runtime.GOOS == "windows" {
		// Check for common Windows tool paths
		windowsPaths := []string{
			"c:/tools",           // Chocolatey default
			"c:/program files/git/bin",
			"c:/program files/git/usr/bin",
			"c:/windows/system32/windowspowershell",
		}
		
		for _, preferred := range windowsPaths {
			if strings.HasPrefix(normalizedDir, preferred) {
				return true
			}
		}
		
		// Check for user-specific preferred paths
		if strings.Contains(normalizedDir, "/appdata/local/programs") &&
			!strings.Contains(normalizedDir, "python") &&
			!strings.Contains(normalizedDir, "node") {
			return true
		}
		
		if strings.Contains(normalizedDir, "/appdata/local/microsoft/windowsapps") {
			return true
		}
	} else {
		// Unix-like preferred paths
		preferredPaths := []string{
			"/usr/local/bin",
			"/opt/homebrew/bin",
			"/snap/bin",
			"/opt/local/bin",  // MacPorts
		}
		
		for _, preferred := range preferredPaths {
			if dir == preferred {
				return true
			}
		}
	}
	
	return false
}

// isLanguageSpecificPath checks if a path is language/package-manager specific
func isLanguageSpecificPath(dir string) bool {
	// Normalize path for comparison
	dir = strings.ToLower(filepath.ToSlash(dir))
	
	// Python-specific paths
	if strings.Contains(dir, "python") {
		// Windows Python Scripts directory
		if strings.Contains(dir, "python") && strings.Contains(dir, "scripts") {
			return true
		}
		// Unix Python site-packages
		if strings.Contains(dir, "site-packages") {
			return true
		}
		// General Python paths
		if strings.Contains(dir, "/python") || strings.Contains(dir, "\\python") {
			return true
		}
	}
	
	// Conda/Anaconda
	if strings.Contains(dir, "conda") || strings.Contains(dir, "anaconda") {
		return true
	}
	
	// Node.js/npm paths
	if strings.Contains(dir, "node_modules") || strings.Contains(dir, "npm") {
		return true
	}
	if strings.Contains(dir, "nodejs") {
		return true
	}
	
	// Ruby/gem paths
	if strings.Contains(dir, "/gems/") || strings.Contains(dir, "/ruby/") {
		return true
	}
	if strings.Contains(dir, "/.gem/") {
		return true
	}
	
	// Rust/cargo paths
	if strings.Contains(dir, "/.cargo/bin") || strings.Contains(dir, "\\.cargo\\bin") {
		return true
	}
	
	// Go paths (but not system Go)
	if strings.Contains(dir, "/go/bin") && !strings.Contains(dir, "/usr/local/go/bin") {
		return true
	}
	
	// Virtual environments
	if strings.Contains(dir, "/venv/") || strings.Contains(dir, "/virtualenv/") {
		return true
	}
	if strings.Contains(dir, "\\venv\\") || strings.Contains(dir, "\\virtualenv\\") {
		return true
	}
	
	// Package managers in user home
	if strings.Contains(dir, "/.local/share/") && (strings.Contains(dir, "/pip/") || 
		strings.Contains(dir, "/pipx/") || strings.Contains(dir, "/poetry/")) {
		return true
	}
	
	// pipx paths
	if strings.Contains(dir, "pipx") {
		return true
	}
	
	return false
}

// isIDESpecificPath checks if a path is IDE-specific and should be avoided
func isIDESpecificPath(normalizedPath string) bool {
	// Common IDE paths to avoid
	idePatterns := []string{
		// VSCode and variants
		".vscode",
		"code/extensions",
		"code/user",
		"code - insiders",
		// Cursor
		"cursor",
		".cursor",
		// JetBrains IDEs
		"jetbrains",
		"intellij",
		".intellij",
		".pycharm",
		"pycharm",
		".webstorm",
		"webstorm",
		".goland",
		"goland",
		".rider",
		"rider",
		// Sublime Text
		"sublime text",
		".sublime",
		// Atom
		".atom",
		// Visual Studio
		"microsoft visual studio",
		"visual studio",
		// Other editors
		"notepad++",
		"brackets",
	}
	
	for _, pattern := range idePatterns {
		if strings.Contains(normalizedPath, pattern) {
			return true
		}
	}
	
	return false
}