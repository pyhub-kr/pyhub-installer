package install

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNewInstaller(t *testing.T) {
	installer := NewInstaller("/source/file", "/dest/file", "755")
	
	if installer.SourcePath != "/source/file" {
		t.Errorf("Expected SourcePath to be /source/file, got %s", installer.SourcePath)
	}
	
	if installer.DestPath != "/dest/file" {
		t.Errorf("Expected DestPath to be /dest/file, got %s", installer.DestPath)
	}
	
	if installer.Chmod != "755" {
		t.Errorf("Expected Chmod to be 755, got %s", installer.Chmod)
	}
}

func TestInstall(t *testing.T) {
	// Skip on Windows for permission tests
	if runtime.GOOS == "windows" {
		t.Skip("Skipping permission tests on Windows")
	}
	
	tempDir, err := os.MkdirTemp("", "install_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create source file
	sourceFile := filepath.Join(tempDir, "source.txt")
	content := []byte("Hello, World!")
	if err := os.WriteFile(sourceFile, content, 0644); err != nil {
		t.Fatal(err)
	}
	
	// Install file
	destFile := filepath.Join(tempDir, "dest", "output.txt")
	installer := NewInstaller(sourceFile, destFile, "755")
	
	if err := installer.Install(); err != nil {
		t.Fatalf("Install failed: %v", err)
	}
	
	// Verify file was copied
	if _, err := os.Stat(destFile); err != nil {
		t.Errorf("Destination file not found: %v", err)
	}
	
	// Verify content
	destContent, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatal(err)
	}
	
	if string(destContent) != string(content) {
		t.Errorf("Expected content %s, got %s", content, destContent)
	}
	
	// Verify permissions (Unix only)
	info, err := os.Stat(destFile)
	if err != nil {
		t.Fatal(err)
	}
	
	expectedMode := os.FileMode(0755)
	if info.Mode().Perm() != expectedMode {
		t.Errorf("Expected permissions %v, got %v", expectedMode, info.Mode().Perm())
	}
}

func TestInstallDirectory(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "install_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create source directory structure
	sourceDir := filepath.Join(tempDir, "source")
	os.MkdirAll(filepath.Join(sourceDir, "subdir"), 0755)
	
	files := map[string]string{
		"file1.txt":        "Content 1",
		"file2.txt":        "Content 2",
		"subdir/file3.txt": "Content 3",
	}
	
	for name, content := range files {
		filePath := filepath.Join(sourceDir, name)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	
	// Install directory
	destDir := filepath.Join(tempDir, "dest")
	installer := NewInstaller(sourceDir, destDir, "755")
	
	if err := installer.InstallDirectory(); err != nil {
		t.Fatalf("InstallDirectory failed: %v", err)
	}
	
	// Verify all files were copied
	for name, expectedContent := range files {
		destFile := filepath.Join(destDir, name)
		content, err := os.ReadFile(destFile)
		if err != nil {
			t.Errorf("Failed to read %s: %v", name, err)
			continue
		}
		
		if string(content) != expectedContent {
			t.Errorf("File %s: expected content %s, got %s", name, expectedContent, content)
		}
	}
	
	// Verify directory structure
	if _, err := os.Stat(filepath.Join(destDir, "subdir")); err != nil {
		t.Error("Subdirectory not created")
	}
}

func TestParseChmod(t *testing.T) {
	installer := &Installer{}
	
	tests := []struct {
		name     string
		chmod    string
		wantMode os.FileMode
		wantErr  bool
	}{
		{
			name:     "Octal 755",
			chmod:    "755",
			wantMode: 0755,
			wantErr:  false,
		},
		{
			name:     "Octal 644",
			chmod:    "644",
			wantMode: 0644,
			wantErr:  false,
		},
		{
			name:     "Octal 777",
			chmod:    "777",
			wantMode: 0777,
			wantErr:  false,
		},
		{
			name:     "Symbolic rwxr-xr-x",
			chmod:    "rwxr-xr-x",
			wantMode: 0755,
			wantErr:  false,
		},
		{
			name:     "Symbolic rw-r--r--",
			chmod:    "rw-r--r--",
			wantMode: 0644,
			wantErr:  false,
		},
		{
			name:     "Symbolic rwxrwxrwx",
			chmod:    "rwxrwxrwx",
			wantMode: 0777,
			wantErr:  false,
		},
		{
			name:    "Invalid format",
			chmod:   "12345",
			wantErr: true,
		},
		{
			name:    "Invalid octal",
			chmod:   "999",
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode, err := installer.parseChmod(tt.chmod)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("parseChmod() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if !tt.wantErr && mode != tt.wantMode {
				t.Errorf("Expected mode %v, got %v", tt.wantMode, mode)
			}
		})
	}
}

func TestParseSymbolicMode(t *testing.T) {
	installer := &Installer{}
	
	tests := []struct {
		name     string
		mode     string
		wantMode os.FileMode
	}{
		{
			name:     "All permissions",
			mode:     "rwxrwxrwx",
			wantMode: 0777,
		},
		{
			name:     "Read only",
			mode:     "r--r--r--",
			wantMode: 0444,
		},
		{
			name:     "Owner only",
			mode:     "rwx------",
			wantMode: 0700,
		},
		{
			name:     "No permissions",
			mode:     "---------",
			wantMode: 0000,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mode, err := installer.parseSymbolicMode(tt.mode)
			if err != nil {
				t.Fatalf("parseSymbolicMode() error = %v", err)
			}
			
			if mode != tt.wantMode {
				t.Errorf("Expected mode %v, got %v", tt.wantMode, mode)
			}
		})
	}
}

func TestIsExecutable(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "install_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	
	installer := &Installer{}
	
	if runtime.GOOS == "windows" {
		// Test Windows executable extensions
		exeFile := filepath.Join(tempDir, "test.exe")
		batFile := filepath.Join(tempDir, "test.bat")
		txtFile := filepath.Join(tempDir, "test.txt")
		
		for _, f := range []string{exeFile, batFile, txtFile} {
			if err := os.WriteFile(f, []byte("test"), 0644); err != nil {
				t.Fatal(err)
			}
		}
		
		exeInfo, _ := os.Stat(exeFile)
		batInfo, _ := os.Stat(batFile)
		txtInfo, _ := os.Stat(txtFile)
		
		if !installer.isExecutable(exeFile, exeInfo) {
			t.Error("Expected .exe to be executable")
		}
		if !installer.isExecutable(batFile, batInfo) {
			t.Error("Expected .bat to be executable")
		}
		if installer.isExecutable(txtFile, txtInfo) {
			t.Error("Expected .txt to not be executable")
		}
	} else {
		// Test Unix executable permissions
		execFile := filepath.Join(tempDir, "executable")
		nonExecFile := filepath.Join(tempDir, "non-executable")
		
		if err := os.WriteFile(execFile, []byte("#!/bin/sh\necho test"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(nonExecFile, []byte("test"), 0644); err != nil {
			t.Fatal(err)
		}
		
		execInfo, _ := os.Stat(execFile)
		nonExecInfo, _ := os.Stat(nonExecFile)
		
		if !installer.isExecutable(execFile, execInfo) {
			t.Error("Expected file with 0755 to be executable")
		}
		if installer.isExecutable(nonExecFile, nonExecInfo) {
			t.Error("Expected file with 0644 to not be executable")
		}
	}
}

func TestFindExecutables(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "install_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create test files
	if runtime.GOOS == "windows" {
		os.WriteFile(filepath.Join(tempDir, "app.exe"), []byte("test"), 0644)
		os.WriteFile(filepath.Join(tempDir, "script.bat"), []byte("test"), 0644)
		os.WriteFile(filepath.Join(tempDir, "data.txt"), []byte("test"), 0644)
	} else {
		os.WriteFile(filepath.Join(tempDir, "app"), []byte("#!/bin/sh"), 0755)
		os.WriteFile(filepath.Join(tempDir, "script.sh"), []byte("#!/bin/sh"), 0755)
		os.WriteFile(filepath.Join(tempDir, "data.txt"), []byte("test"), 0644)
	}
	
	executables, err := FindExecutables(tempDir)
	if err != nil {
		t.Fatalf("FindExecutables failed: %v", err)
	}
	
	if len(executables) != 2 {
		t.Errorf("Expected 2 executables, found %d", len(executables))
	}
	
	// Verify we found the right files
	foundApp := false
	foundScript := false
	for _, exec := range executables {
		base := filepath.Base(exec)
		if runtime.GOOS == "windows" {
			if base == "app.exe" {
				foundApp = true
			}
			if base == "script.bat" {
				foundScript = true
			}
		} else {
			if base == "app" {
				foundApp = true
			}
			if base == "script.sh" {
				foundScript = true
			}
		}
	}
	
	if !foundApp || !foundScript {
		t.Error("Did not find expected executables")
	}
}

func TestAddToPath(t *testing.T) {
	// Just test that the function runs without error
	// Actual PATH modification is not implemented
	err := AddToPath("/test/path")
	
	// Currently returns nil for all platforms
	if err != nil {
		t.Errorf("AddToPath returned unexpected error: %v", err)
	}
}

func TestCopyFile(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "install_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create source file with specific content
	sourceFile := filepath.Join(tempDir, "source.bin")
	content := make([]byte, 1024)
	for i := range content {
		content[i] = byte(i % 256)
	}
	
	if err := os.WriteFile(sourceFile, content, 0644); err != nil {
		t.Fatal(err)
	}
	
	// Copy file
	destFile := filepath.Join(tempDir, "dest.bin")
	installer := NewInstaller(sourceFile, destFile, "")
	
	if err := installer.copyFile(); err != nil {
		t.Fatalf("copyFile failed: %v", err)
	}
	
	// Verify content
	destContent, err := os.ReadFile(destFile)
	if err != nil {
		t.Fatal(err)
	}
	
	if len(destContent) != len(content) {
		t.Errorf("Expected %d bytes, got %d", len(content), len(destContent))
	}
	
	for i := range content {
		if destContent[i] != content[i] {
			t.Errorf("Content mismatch at byte %d", i)
			break
		}
	}
}

// TestFindWritableInstallPath tests finding writable install path
func TestFindWritableInstallPath(t *testing.T) {
	// This test checks that the function returns a path
	path, err := FindWritableInstallPath()
	if err != nil {
		t.Fatalf("FindWritableInstallPath failed: %v", err)
	}
	
	if path == "" {
		t.Error("FindWritableInstallPath returned empty path")
	}
	
	// Test that the returned path is actually writable
	if !isDirectoryWritable(path) {
		t.Errorf("Returned path is not writable: %s", path)
	}
	
	t.Logf("Found writable install path: %s", path)
}

// TestGetPathDirectories tests PATH parsing
func TestGetPathDirectories(t *testing.T) {
	// Save original PATH
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)
	
	// Test with custom PATH
	testPath := "/usr/local/bin:/usr/bin:/bin"
	if runtime.GOOS == "windows" {
		testPath = "C:\\bin;C:\\Windows\\system32"
	}
	
	os.Setenv("PATH", testPath)
	
	dirs := getPathDirectories()
	if len(dirs) == 0 {
		t.Error("getPathDirectories returned no directories")
	}
	
	t.Logf("PATH directories: %v", dirs)
}

// TestIsDirectoryWritable tests directory writability check
func TestIsDirectoryWritable(t *testing.T) {
	// Test with temp directory (should be writable)
	tempDir := t.TempDir()
	if !isDirectoryWritable(tempDir) {
		t.Errorf("Temp directory should be writable: %s", tempDir)
	}
	
	// Test with non-existent directory
	if isDirectoryWritable("/non/existent/path") {
		t.Error("Non-existent directory should not be writable")
	}
	
	// Test with system directory (likely not writable for regular users)
	if runtime.GOOS != "windows" {
		writable := isDirectoryWritable("/root")
		t.Logf("/root writable: %v", writable)
	}
}

// TestGetFallbackDirectories tests fallback directory generation
func TestGetFallbackDirectories(t *testing.T) {
	fallbacks := getFallbackDirectories()
	if len(fallbacks) == 0 {
		t.Error("getFallbackDirectories returned no directories")
	}
	
	// Should include user home-based directories
	homeDir, err := os.UserHomeDir()
	if err == nil {
		found := false
		for _, dir := range fallbacks {
			if strings.HasPrefix(dir, homeDir) {
				found = true
				break
			}
		}
		if !found {
			t.Error("No home-based directories in fallbacks")
		}
	}
	
	t.Logf("Fallback directories: %v", fallbacks)
}

// TestIsLanguageSpecificPath tests language-specific path detection
func TestIsLanguageSpecificPath(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
		desc     string
	}{
		// Python paths
		{`C:\Users\user\AppData\Local\Programs\Python\Python312\Scripts`, true, "Windows Python Scripts"},
		{`/usr/local/lib/python3.9/site-packages`, true, "Unix Python site-packages"},
		{`/home/user/.local/lib/python3.8/site-packages`, true, "User Python packages"},
		{`/opt/anaconda3/bin`, true, "Anaconda path"},
		
		// Node.js paths
		{`/usr/local/lib/node_modules/.bin`, true, "Global npm bin"},
		{`/home/user/.npm-global/bin`, true, "User npm global"},
		{`C:\Users\user\AppData\Roaming\npm`, true, "Windows npm"},
		
		// Ruby paths
		{`/usr/local/lib/ruby/gems/3.0.0/bin`, true, "Ruby gems"},
		{`/home/user/.gem/ruby/2.7.0/bin`, true, "User Ruby gems"},
		
		// Other language paths
		{`/home/user/.cargo/bin`, true, "Rust cargo"},
		{`/home/user/go/bin`, true, "Go user bin"},
		{`/home/user/project/venv/bin`, true, "Python virtualenv"},
		
		// Non-language paths
		{`/usr/local/bin`, false, "System bin"},
		{`/usr/bin`, false, "System bin"},
		{`C:\Windows\System32`, false, "Windows system"},
		{`C:\tools`, false, "Tools directory"},
		{`/opt/homebrew/bin`, false, "Homebrew"},
		{`/home/user/bin`, false, "User bin"},
		{`/usr/local/go/bin`, false, "System Go installation"},
	}
	
	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			result := isLanguageSpecificPath(tt.path)
			if result != tt.expected {
				t.Errorf("isLanguageSpecificPath(%s) = %v, want %v", tt.path, result, tt.expected)
			}
		})
	}
}

// TestPathPrioritization tests that paths are prioritized correctly
func TestPathPrioritization(t *testing.T) {
	// Save original PATH
	originalPath := os.Getenv("PATH")
	defer os.Setenv("PATH", originalPath)
	
	// Create test PATH with various types of directories
	separator := ":"
	if runtime.GOOS == "windows" {
		separator = ";"
	}
	
	homeDir, _ := os.UserHomeDir()
	var testPaths []string
	
	if runtime.GOOS == "windows" {
		testPaths = []string{
			`C:\Users\user\AppData\Local\Programs\Python\Python312\Scripts`,
			`C:\Users\user\AppData\Local\Programs`,
			`C:\Users\user\bin`,
			`C:\tools`,
			`C:\Program Files\nodejs`,
		}
	} else {
		testPaths = []string{
			filepath.Join(homeDir, ".local", "lib", "python3.9", "site-packages", "bin"), // Python path
			filepath.Join(homeDir, ".cargo", "bin"),                                       // Rust path
			filepath.Join(homeDir, ".local", "bin"),                                       // User local bin
			filepath.Join(homeDir, "bin"),                                                 // User bin
			"/usr/local/bin",                                                              // System bin
			filepath.Join(homeDir, "node_modules", ".bin"),                               // Node.js path
		}
	}
	
	os.Setenv("PATH", strings.Join(testPaths, separator))
	
	// Get prioritized paths
	paths := getPathDirectories()
	
	// Find indices of different path types
	pythonIndex := -1
	nodeIndex := -1
	userBinIndex := -1
	
	for i, path := range paths {
		normalizedPath := strings.ToLower(filepath.ToSlash(path))
		if strings.Contains(normalizedPath, "python") && strings.Contains(normalizedPath, "scripts") {
			pythonIndex = i
		} else if strings.Contains(normalizedPath, "node") {
			nodeIndex = i
		} else if (strings.Contains(normalizedPath, "/bin") || strings.Contains(normalizedPath, "\\bin")) && 
			!strings.Contains(normalizedPath, "python") && 
			!strings.Contains(normalizedPath, "node") &&
			!strings.Contains(normalizedPath, "cargo") {
			if userBinIndex == -1 {
				userBinIndex = i
			}
		}
	}
	
	// User bin should come before language-specific paths
	if pythonIndex != -1 && userBinIndex != -1 {
		if userBinIndex > pythonIndex {
			t.Errorf("User bin (index %d) should come before Python path (index %d)", userBinIndex, pythonIndex)
		}
	}
	if nodeIndex != -1 && userBinIndex != -1 {
		if userBinIndex > nodeIndex {
			t.Errorf("User bin (index %d) should come before Node path (index %d)", userBinIndex, nodeIndex)
		}
	}
	
	t.Logf("Path priorities - User bin: %d, Python: %d, Node: %d", userBinIndex, pythonIndex, nodeIndex)
	t.Logf("All paths: %v", paths)
}