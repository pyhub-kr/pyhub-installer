package extract

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Extractor handles archive extraction
type Extractor struct {
	ArchivePath string
	DestPath    string
	flatten     bool
	autoFlatten bool
}

// NewExtractor creates a new extractor
func NewExtractor(archivePath, destPath string) *Extractor {
	return &Extractor{
		ArchivePath: archivePath,
		DestPath:    destPath,
		flatten:     false,
		autoFlatten: false,
	}
}

// SetFlatten enables or disables flattening
func (e *Extractor) SetFlatten(flatten bool) {
	e.flatten = flatten
}

// SetAutoFlatten enables or disables auto-flattening
func (e *Extractor) SetAutoFlatten(autoFlatten bool) {
	e.autoFlatten = autoFlatten
}

// Extract extracts archive based on file extension
func (e *Extractor) Extract() error {
	// Create destination directory
	if err := os.MkdirAll(e.DestPath, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(e.ArchivePath))
	
	switch ext {
	case ".zip":
		return e.extractZip()
	case ".gz":
		if strings.HasSuffix(strings.ToLower(e.ArchivePath), ".tar.gz") {
			return e.extractTarGz()
		}
		return e.extractGzip()
	case ".tar":
		return e.extractTar()
	default:
		return fmt.Errorf("unsupported archive format: %s", ext)
	}
}

// extractZip extracts ZIP archives
func (e *Extractor) extractZip() error {
	reader, err := zip.OpenReader(e.ArchivePath)
	if err != nil {
		return fmt.Errorf("failed to open ZIP file: %w", err)
	}
	defer reader.Close()

	fmt.Printf("Extracting ZIP archive to %s...\n", e.DestPath)

	// Detect top-level directories if auto-flatten is enabled
	topDirs, _ := e.detectTopLevelDirsZip(&reader.Reader)
	shouldFlatten := e.shouldFlatten(topDirs)
	
	if shouldFlatten && len(topDirs) == 1 {
		for dir := range topDirs {
			fmt.Printf("Flattening: removing top-level directory '%s'\n", dir)
			break
		}
	}

	for _, file := range reader.File {
		if err := e.extractZipFile(file, shouldFlatten); err != nil {
			return fmt.Errorf("failed to extract %s: %w", file.Name, err)
		}
	}

	fmt.Println("✓ ZIP extraction completed")
	return nil
}

// extractZipFile extracts a single file from ZIP
func (e *Extractor) extractZipFile(file *zip.File, shouldFlatten bool) error {
	// Apply flattening if needed
	fileName := file.Name
	if shouldFlatten {
		fileName = stripTopLevel(fileName)
		if fileName == "" {
			return nil // Skip the top-level directory itself
		}
	}
	
	// Security check: prevent zip slip
	destPath := filepath.Join(e.DestPath, fileName)
	if !strings.HasPrefix(destPath, filepath.Clean(e.DestPath)+string(os.PathSeparator)) {
		return fmt.Errorf("invalid file path: %s", file.Name)
	}

	if file.FileInfo().IsDir() {
		return os.MkdirAll(destPath, file.FileInfo().Mode())
	}

	// Create directory for file
	if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
		return err
	}

	// Extract file
	reader, err := file.Open()
	if err != nil {
		return err
	}
	defer reader.Close()

	writer, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.FileInfo().Mode())
	if err != nil {
		return err
	}
	defer writer.Close()

	_, err = io.Copy(writer, reader)
	return err
}

// extractTarGz extracts TAR.GZ archives
func (e *Extractor) extractTarGz() error {
	file, err := os.Open(e.ArchivePath)
	if err != nil {
		return fmt.Errorf("failed to open TAR.GZ file: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	fmt.Printf("Extracting TAR.GZ archive to %s...\n", e.DestPath)

	// For tar.gz, we can't easily seek, so we'll read it twice if needed
	if e.flatten || e.autoFlatten {
		// First pass: detect top-level directories
		tarReader := tar.NewReader(gzReader)
		topDirs, _ := e.detectTopLevelDirsTar(tarReader)
		shouldFlatten := e.shouldFlatten(topDirs)
		
		if shouldFlatten && len(topDirs) == 1 {
			for dir := range topDirs {
				fmt.Printf("Flattening: removing top-level directory '%s'\n", dir)
				break
			}
		}
		
		// Re-open file for second pass
		file.Close()
		gzReader.Close()
		
		file, err = os.Open(e.ArchivePath)
		if err != nil {
			return fmt.Errorf("failed to reopen TAR.GZ file: %w", err)
		}
		defer file.Close()
		
		gzReader, err = gzip.NewReader(file)
		if err != nil {
			return fmt.Errorf("failed to recreate gzip reader: %w", err)
		}
		defer gzReader.Close()
		
		tarReader = tar.NewReader(gzReader)
		return e.extractTarReaderWithFlatten(tarReader, shouldFlatten)
	}

	tarReader := tar.NewReader(gzReader)
	return e.extractTarReader(tarReader)
}

// extractTar extracts TAR archives
func (e *Extractor) extractTar() error {
	file, err := os.Open(e.ArchivePath)
	if err != nil {
		return fmt.Errorf("failed to open TAR file: %w", err)
	}
	defer file.Close()

	fmt.Printf("Extracting TAR archive to %s...\n", e.DestPath)

	return e.extractTarWithFlatten(file)
}

// extractTarReader extracts from tar reader
func (e *Extractor) extractTarReader(tarReader *tar.Reader) error {
	return e.extractTarReaderWithFlatten(tarReader, false)
}

// extractTarWithFlatten handles TAR extraction with flatten support
func (e *Extractor) extractTarWithFlatten(file *os.File) error {
	// First pass: detect top-level directories if needed
	var topDirs map[string]bool
	var shouldFlatten bool
	
	if e.flatten || e.autoFlatten {
		file.Seek(0, 0)
		tarReader := tar.NewReader(file)
		topDirs, _ = e.detectTopLevelDirsTar(tarReader)
		shouldFlatten = e.shouldFlatten(topDirs)
		
		if shouldFlatten && len(topDirs) == 1 {
			for dir := range topDirs {
				fmt.Printf("Flattening: removing top-level directory '%s'\n", dir)
				break
			}
		}
		
		// Reset file position for second pass
		file.Seek(0, 0)
	}
	
	// Second pass: extract files
	tarReader := tar.NewReader(file)
	return e.extractTarReaderWithFlatten(tarReader, shouldFlatten)
}

// extractTarReaderWithFlatten extracts from tar reader with optional flattening
func (e *Extractor) extractTarReaderWithFlatten(tarReader *tar.Reader, shouldFlatten bool) error {
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		if err := e.extractTarFile(header, tarReader, shouldFlatten); err != nil {
			return fmt.Errorf("failed to extract %s: %w", header.Name, err)
		}
	}

	fmt.Println("✓ TAR extraction completed")
	return nil
}

// extractTarFile extracts a single file from TAR
func (e *Extractor) extractTarFile(header *tar.Header, reader *tar.Reader, shouldFlatten bool) error {
	// Apply flattening if needed
	fileName := header.Name
	if shouldFlatten {
		fileName = stripTopLevel(fileName)
		if fileName == "" {
			return nil // Skip the top-level directory itself
		}
	}
	
	// Security check: prevent tar slip
	destPath := filepath.Join(e.DestPath, fileName)
	if !strings.HasPrefix(destPath, filepath.Clean(e.DestPath)+string(os.PathSeparator)) {
		return fmt.Errorf("invalid file path: %s", header.Name)
	}

	switch header.Typeflag {
	case tar.TypeDir:
		return os.MkdirAll(destPath, os.FileMode(header.Mode))
	case tar.TypeReg:
		// Create directory for file
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		// Extract file
		writer, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
		if err != nil {
			return err
		}
		defer writer.Close()

		_, err = io.Copy(writer, reader)
		return err
	default:
		// Skip unsupported file types (symlinks, etc.)
		return nil
	}
}

// extractGzip extracts single GZIP files
func (e *Extractor) extractGzip() error {
	file, err := os.Open(e.ArchivePath)
	if err != nil {
		return fmt.Errorf("failed to open GZIP file: %w", err)
	}
	defer file.Close()

	gzReader, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()

	// Determine output filename
	outputName := strings.TrimSuffix(filepath.Base(e.ArchivePath), ".gz")
	outputPath := filepath.Join(e.DestPath, outputName)

	writer, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer writer.Close()

	fmt.Printf("Extracting GZIP file to %s...\n", outputPath)

	_, err = io.Copy(writer, gzReader)
	if err != nil {
		return fmt.Errorf("failed to extract GZIP: %w", err)
	}

	fmt.Println("✓ GZIP extraction completed")
	return nil
}

// shouldFlatten determines if extraction should be flattened
func (e *Extractor) shouldFlatten(topLevelDirs map[string]bool) bool {
	if e.flatten {
		return true
	}
	if e.autoFlatten && len(topLevelDirs) == 1 {
		return true
	}
	return false
}

// detectTopLevelDirs detects top-level directories in a ZIP archive
func (e *Extractor) detectTopLevelDirsZip(reader *zip.Reader) (map[string]bool, error) {
	topDirs := make(map[string]bool)
	
	for _, file := range reader.File {
		parts := strings.Split(file.Name, "/")
		if len(parts) > 0 && parts[0] != "" {
			topDirs[parts[0]] = true
		}
	}
	
	return topDirs, nil
}

// detectTopLevelDirsTar detects top-level directories in a TAR archive
func (e *Extractor) detectTopLevelDirsTar(tarReader *tar.Reader) (map[string]bool, error) {
	topDirs := make(map[string]bool)
	
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		
		parts := strings.Split(header.Name, "/")
		if len(parts) > 0 && parts[0] != "" {
			topDirs[parts[0]] = true
		}
	}
	
	return topDirs, nil
}

// stripTopLevel removes the top-level directory from a path
func stripTopLevel(path string) string {
	parts := strings.Split(path, "/")
	if len(parts) > 1 {
		return strings.Join(parts[1:], "/")
	}
	return ""
}