package extract

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewExtractor(t *testing.T) {
	e := NewExtractor("/path/to/archive.zip", "/path/to/dest")
	
	if e.ArchivePath != "/path/to/archive.zip" {
		t.Errorf("Expected ArchivePath to be /path/to/archive.zip, got %s", e.ArchivePath)
	}
	
	if e.DestPath != "/path/to/dest" {
		t.Errorf("Expected DestPath to be /path/to/dest, got %s", e.DestPath)
	}
}

func TestExtractUnsupportedFormat(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "extract_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create a dummy file with unsupported extension
	unsupportedFile := filepath.Join(tempDir, "test.xyz")
	if err := os.WriteFile(unsupportedFile, []byte("test"), 0644); err != nil {
		t.Fatal(err)
	}
	
	e := NewExtractor(unsupportedFile, tempDir)
	err = e.Extract()
	
	if err == nil {
		t.Error("Expected error for unsupported format, got nil")
	}
	
	if err != nil && !contains(err.Error(), "unsupported archive format") {
		t.Errorf("Expected unsupported format error, got: %v", err)
	}
}

func TestExtractZip(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "extract_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create a test ZIP file
	zipFile := filepath.Join(tempDir, "test.zip")
	if err := createTestZip(zipFile); err != nil {
		t.Fatal(err)
	}
	
	// Extract ZIP
	destDir := filepath.Join(tempDir, "extracted")
	e := NewExtractor(zipFile, destDir)
	
	if err := e.Extract(); err != nil {
		t.Fatalf("Failed to extract ZIP: %v", err)
	}
	
	// Verify extracted files
	verifyExtractedFiles(t, destDir)
}

func TestExtractTar(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "extract_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create a test TAR file
	tarFile := filepath.Join(tempDir, "test.tar")
	if err := createTestTar(tarFile, false); err != nil {
		t.Fatal(err)
	}
	
	// Extract TAR
	destDir := filepath.Join(tempDir, "extracted")
	e := NewExtractor(tarFile, destDir)
	
	if err := e.Extract(); err != nil {
		t.Fatalf("Failed to extract TAR: %v", err)
	}
	
	// Verify extracted files
	verifyExtractedFiles(t, destDir)
}

func TestExtractTarGz(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "extract_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create a test TAR.GZ file
	tarGzFile := filepath.Join(tempDir, "test.tar.gz")
	if err := createTestTar(tarGzFile, true); err != nil {
		t.Fatal(err)
	}
	
	// Extract TAR.GZ
	destDir := filepath.Join(tempDir, "extracted")
	e := NewExtractor(tarGzFile, destDir)
	
	if err := e.Extract(); err != nil {
		t.Fatalf("Failed to extract TAR.GZ: %v", err)
	}
	
	// Verify extracted files
	verifyExtractedFiles(t, destDir)
}

func TestExtractGzip(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "extract_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create a test GZIP file
	gzFile := filepath.Join(tempDir, "test.txt.gz")
	content := []byte("This is a test content for gzip extraction.")
	
	file, err := os.Create(gzFile)
	if err != nil {
		t.Fatal(err)
	}
	
	gzWriter := gzip.NewWriter(file)
	if _, err := gzWriter.Write(content); err != nil {
		t.Fatal(err)
	}
	gzWriter.Close()
	file.Close()
	
	// Extract GZIP
	destDir := filepath.Join(tempDir, "extracted")
	e := NewExtractor(gzFile, destDir)
	
	if err := e.Extract(); err != nil {
		t.Fatalf("Failed to extract GZIP: %v", err)
	}
	
	// Verify extracted file
	extractedFile := filepath.Join(destDir, "test.txt")
	extractedContent, err := os.ReadFile(extractedFile)
	if err != nil {
		t.Fatal(err)
	}
	
	if string(extractedContent) != string(content) {
		t.Errorf("Expected content %s, got %s", content, extractedContent)
	}
}

func TestZipSlipPrevention(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "extract_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create a malicious ZIP with path traversal
	zipFile := filepath.Join(tempDir, "malicious.zip")
	
	file, err := os.Create(zipFile)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	
	zipWriter := zip.NewWriter(file)
	
	// Add file with path traversal attempt
	writer, err := zipWriter.Create("../../../etc/passwd")
	if err != nil {
		t.Fatal(err)
	}
	writer.Write([]byte("malicious content"))
	zipWriter.Close()
	
	// Try to extract
	destDir := filepath.Join(tempDir, "safe")
	e := NewExtractor(zipFile, destDir)
	
	err = e.Extract()
	if err == nil {
		t.Error("Expected error for zip slip attempt, got nil")
	}
}

func TestTarSlipPrevention(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "extract_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create a malicious TAR with path traversal
	tarFile := filepath.Join(tempDir, "malicious.tar")
	
	file, err := os.Create(tarFile)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	
	tarWriter := tar.NewWriter(file)
	
	// Add file with path traversal attempt
	header := &tar.Header{
		Name: "../../../etc/passwd",
		Mode: 0644,
		Size: int64(len("malicious content")),
	}
	
	if err := tarWriter.WriteHeader(header); err != nil {
		t.Fatal(err)
	}
	
	if _, err := tarWriter.Write([]byte("malicious content")); err != nil {
		t.Fatal(err)
	}
	
	tarWriter.Close()
	
	// Try to extract
	destDir := filepath.Join(tempDir, "safe")
	e := NewExtractor(tarFile, destDir)
	
	err = e.Extract()
	if err == nil {
		t.Error("Expected error for tar slip attempt, got nil")
	}
}

// Helper functions

func createTestZip(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()
	
	// Add directory with proper mode
	header := &zip.FileHeader{
		Name: "testdir/",
		Method: zip.Deflate,
	}
	header.SetMode(os.FileMode(0755) | os.ModeDir)
	_, err = zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}
	
	// Add files
	files := map[string]string{
		"file1.txt":        "Content of file 1",
		"file2.txt":        "Content of file 2",
		"testdir/file3.txt": "Content of file 3 in directory",
	}
	
	for name, content := range files {
		header := &zip.FileHeader{
			Name: name,
			Method: zip.Deflate,
		}
		header.SetMode(0644)
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}
		_, err = writer.Write([]byte(content))
		if err != nil {
			return err
		}
	}
	
	return nil
}

func createTestTar(filename string, compress bool) error {
	var buf bytes.Buffer
	tarWriter := tar.NewWriter(&buf)
	
	// Add directory
	dirHeader := &tar.Header{
		Name:     "testdir/",
		Mode:     0755,
		Typeflag: tar.TypeDir,
	}
	if err := tarWriter.WriteHeader(dirHeader); err != nil {
		return err
	}
	
	// Add files
	files := map[string]string{
		"file1.txt":        "Content of file 1",
		"file2.txt":        "Content of file 2",
		"testdir/file3.txt": "Content of file 3 in directory",
	}
	
	for name, content := range files {
		header := &tar.Header{
			Name: name,
			Mode: 0644,
			Size: int64(len(content)),
		}
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if _, err := tarWriter.Write([]byte(content)); err != nil {
			return err
		}
	}
	
	if err := tarWriter.Close(); err != nil {
		return err
	}
	
	// Write to file
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()
	
	if compress {
		gzWriter := gzip.NewWriter(file)
		defer gzWriter.Close()
		_, err = io.Copy(gzWriter, &buf)
	} else {
		_, err = io.Copy(file, &buf)
	}
	
	return err
}

func verifyExtractedFiles(t *testing.T, destDir string) {
	expectedFiles := map[string]string{
		"file1.txt":        "Content of file 1",
		"file2.txt":        "Content of file 2",
		"testdir/file3.txt": "Content of file 3 in directory",
	}
	
	for name, expectedContent := range expectedFiles {
		filePath := filepath.Join(destDir, name)
		content, err := os.ReadFile(filePath)
		if err != nil {
			t.Errorf("Failed to read extracted file %s: %v", name, err)
			continue
		}
		
		if string(content) != expectedContent {
			t.Errorf("File %s: expected content %q, got %q", name, expectedContent, string(content))
		}
	}
	
	// Check directory exists
	dirPath := filepath.Join(destDir, "testdir")
	info, err := os.Stat(dirPath)
	if err != nil {
		t.Errorf("Directory testdir not found: %v", err)
	} else if !info.IsDir() {
		t.Error("testdir is not a directory")
	}
}

func contains(s, substr string) bool {
	return bytes.Contains([]byte(s), []byte(substr))
}

// TestExtractZipWithFlatten tests ZIP extraction with flatten
func TestExtractZipWithFlatten(t *testing.T) {
	// Create temp directories
	tempDir := t.TempDir()
	archivePath := filepath.Join(tempDir, "test-flatten.zip")
	extractPath := filepath.Join(tempDir, "extract")

	// Create ZIP with single top-level directory
	zipFile, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}

	zipWriter := zip.NewWriter(zipFile)
	
	// Add files under a single top-level directory
	files := []struct {
		name    string
		content string
	}{
		{"myapp/README.md", "# My App"},
		{"myapp/main.go", "package main"},
		{"myapp/config/settings.json", "{}"},
	}

	for _, file := range files {
		writer, err := zipWriter.Create(file.name)
		if err != nil {
			t.Fatal(err)
		}
		_, err = writer.Write([]byte(file.content))
		if err != nil {
			t.Fatal(err)
		}
	}

	zipWriter.Close()
	zipFile.Close()

	t.Run("WithoutFlatten", func(t *testing.T) {
		// Clean extract directory
		os.RemoveAll(extractPath)
		
		extractor := NewExtractor(archivePath, extractPath)
		err := extractor.Extract()
		if err != nil {
			t.Fatal(err)
		}

		// Check that top-level directory exists
		if _, err := os.Stat(filepath.Join(extractPath, "myapp", "README.md")); err != nil {
			t.Error("Expected myapp/README.md to exist")
		}
	})

	t.Run("WithFlatten", func(t *testing.T) {
		// Clean extract directory
		os.RemoveAll(extractPath)
		
		extractor := NewExtractor(archivePath, extractPath)
		extractor.SetFlatten(true)
		err := extractor.Extract()
		if err != nil {
			t.Fatal(err)
		}

		// Check that files are extracted without top-level directory
		if _, err := os.Stat(filepath.Join(extractPath, "README.md")); err != nil {
			t.Error("Expected README.md to exist at root level")
		}
		if _, err := os.Stat(filepath.Join(extractPath, "main.go")); err != nil {
			t.Error("Expected main.go to exist at root level")
		}
		if _, err := os.Stat(filepath.Join(extractPath, "config", "settings.json")); err != nil {
			t.Error("Expected config/settings.json to exist")
		}
	})

	t.Run("WithAutoFlatten", func(t *testing.T) {
		// Clean extract directory
		os.RemoveAll(extractPath)
		
		extractor := NewExtractor(archivePath, extractPath)
		extractor.SetAutoFlatten(true)
		err := extractor.Extract()
		if err != nil {
			t.Fatal(err)
		}

		// Should auto-flatten since there's only one top-level directory
		if _, err := os.Stat(filepath.Join(extractPath, "README.md")); err != nil {
			t.Error("Expected README.md to be auto-flattened to root level")
		}
	})
}

// TestExtractTarGzWithFlatten tests TAR.GZ extraction with flatten
func TestExtractTarGzWithFlatten(t *testing.T) {
	// Create temp directories
	tempDir := t.TempDir()
	archivePath := filepath.Join(tempDir, "test-flatten.tar.gz")
	extractPath := filepath.Join(tempDir, "extract")

	// Create TAR.GZ with single top-level directory
	file, err := os.Create(archivePath)
	if err != nil {
		t.Fatal(err)
	}

	gzWriter := gzip.NewWriter(file)
	tarWriter := tar.NewWriter(gzWriter)

	// Add files under a single top-level directory
	files := []struct {
		name    string
		content string
		mode    int64
	}{
		{"myapp/", "", 0755},
		{"myapp/README.md", "# My App", 0644},
		{"myapp/main.go", "package main", 0644},
		{"myapp/config/", "", 0755},
		{"myapp/config/settings.json", "{}", 0644},
	}

	for _, f := range files {
		header := &tar.Header{
			Name: f.name,
			Mode: f.mode,
			Size: int64(len(f.content)),
		}
		
		if strings.HasSuffix(f.name, "/") {
			header.Typeflag = tar.TypeDir
		} else {
			header.Typeflag = tar.TypeReg
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			t.Fatal(err)
		}

		if f.content != "" {
			if _, err := tarWriter.Write([]byte(f.content)); err != nil {
				t.Fatal(err)
			}
		}
	}

	tarWriter.Close()
	gzWriter.Close()
	file.Close()

	t.Run("WithFlatten", func(t *testing.T) {
		// Clean extract directory
		os.RemoveAll(extractPath)
		
		extractor := NewExtractor(archivePath, extractPath)
		extractor.SetFlatten(true)
		err := extractor.Extract()
		if err != nil {
			t.Fatal(err)
		}

		// Check that files are extracted without top-level directory
		if _, err := os.Stat(filepath.Join(extractPath, "README.md")); err != nil {
			t.Error("Expected README.md to exist at root level")
		}
		if _, err := os.Stat(filepath.Join(extractPath, "main.go")); err != nil {
			t.Error("Expected main.go to exist at root level")
		}
		if _, err := os.Stat(filepath.Join(extractPath, "config", "settings.json")); err != nil {
			t.Error("Expected config/settings.json to exist")
		}
	})
}