package download

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewChunkDownloader(t *testing.T) {
	cd := NewChunkDownloader("http://example.com/file.zip", "output.zip")
	
	if cd.URL != "http://example.com/file.zip" {
		t.Errorf("Expected URL to be http://example.com/file.zip, got %s", cd.URL)
	}
	
	if cd.Filename != "output.zip" {
		t.Errorf("Expected Filename to be output.zip, got %s", cd.Filename)
	}
	
	if cd.ChunkSize != 1024*1024 {
		t.Errorf("Expected ChunkSize to be 1MB, got %d", cd.ChunkSize)
	}
	
	if cd.Parallelism != 4 {
		t.Errorf("Expected Parallelism to be 4, got %d", cd.Parallelism)
	}
}

func TestCreateChunks(t *testing.T) {
	cd := NewChunkDownloader("", "")
	cd.ChunkSize = 100
	
	tests := []struct {
		name          string
		contentLength int64
		expectedChunks int
	}{
		{"Small file", 50, 1},
		{"Exact chunk size", 100, 1},
		{"Multiple chunks", 250, 3},
		{"Large file", 1024, 11},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunks := cd.createChunks(tt.contentLength)
			if len(chunks) != tt.expectedChunks {
				t.Errorf("Expected %d chunks, got %d", tt.expectedChunks, len(chunks))
			}
			
			// Verify chunk ranges
			for i, chunk := range chunks {
				if chunk.Index != i {
					t.Errorf("Expected chunk index %d, got %d", i, chunk.Index)
				}
				
				if i == len(chunks)-1 {
					// Last chunk
					if chunk.End != tt.contentLength-1 {
						t.Errorf("Expected last chunk end to be %d, got %d", tt.contentLength-1, chunk.End)
					}
				}
			}
		})
	}
}

func TestDownloadSingle(t *testing.T) {
	// Create test server
	content := []byte("Hello, World! This is test content.")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
		w.WriteHeader(http.StatusOK)
		w.Write(content)
	}))
	defer server.Close()
	
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "download_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	
	outputFile := filepath.Join(tempDir, "output.txt")
	cd := NewChunkDownloader(server.URL, outputFile)
	
	ctx := context.Background()
	err = cd.downloadSingle(ctx)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}
	
	// Verify file content
	downloaded, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatal(err)
	}
	
	if string(downloaded) != string(content) {
		t.Errorf("Expected content %s, got %s", content, downloaded)
	}
}

func TestDownloadWithChunks(t *testing.T) {
	// Create test server that supports range requests
	content := make([]byte, 1024) // 1KB of data
	for i := range content {
		content[i] = byte(i % 256)
	}
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHeader := r.Header.Get("Range")
		if rangeHeader == "" {
			// HEAD request
			w.Header().Set("Accept-Ranges", "bytes")
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
			if r.Method == "HEAD" {
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write(content)
		} else {
			// Parse range header
			var start, end int64
			fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end)
			
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(content)))
			w.Header().Set("Content-Length", fmt.Sprintf("%d", end-start+1))
			w.WriteHeader(http.StatusPartialContent)
			w.Write(content[start : end+1])
		}
	}))
	defer server.Close()
	
	// Create temp directory
	tempDir, err := os.MkdirTemp("", "download_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	
	outputFile := filepath.Join(tempDir, "output.bin")
	cd := NewChunkDownloader(server.URL, outputFile)
	cd.ChunkSize = 256 // Use smaller chunks for testing
	
	ctx := context.Background()
	err = cd.Download(ctx)
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}
	
	// Verify file content
	downloaded, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatal(err)
	}
	
	if len(downloaded) != len(content) {
		t.Errorf("Expected %d bytes, got %d", len(content), len(downloaded))
	}
	
	for i := range content {
		if downloaded[i] != content[i] {
			t.Errorf("Content mismatch at byte %d: expected %d, got %d", i, content[i], downloaded[i])
			break
		}
	}
}

func TestDownloadWithTimeout(t *testing.T) {
	// Create a slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	tempDir, err := os.MkdirTemp("", "download_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	
	outputFile := filepath.Join(tempDir, "output.txt")
	cd := NewChunkDownloader(server.URL, outputFile)
	
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	err = cd.downloadSingle(ctx)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
}

func TestDownloadWithError(t *testing.T) {
	// Create server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()
	
	tempDir, err := os.MkdirTemp("", "download_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	
	outputFile := filepath.Join(tempDir, "output.txt")
	cd := NewChunkDownloader(server.URL, outputFile)
	
	ctx := context.Background()
	err = cd.downloadSingle(ctx)
	if err == nil {
		t.Error("Expected download error, got nil")
	}
}

func TestMergeChunks(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "download_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)
	
	// Create test chunk files
	chunks := [][]byte{
		[]byte("Hello, "),
		[]byte("World! "),
		[]byte("This is a test."),
	}
	
	tempFiles := make([]*os.File, len(chunks))
	for i, chunk := range chunks {
		tempFile, err := os.CreateTemp(tempDir, fmt.Sprintf("chunk_%d_", i))
		if err != nil {
			t.Fatal(err)
		}
		tempFiles[i] = tempFile
		
		if _, err := tempFile.Write(chunk); err != nil {
			t.Fatal(err)
		}
	}
	
	outputFile := filepath.Join(tempDir, "merged.txt")
	cd := NewChunkDownloader("", outputFile)
	
	err = cd.mergeChunks(tempFiles)
	if err != nil {
		t.Fatalf("Merge failed: %v", err)
	}
	
	// Verify merged content
	merged, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatal(err)
	}
	
	expected := "Hello, World! This is a test."
	if string(merged) != expected {
		t.Errorf("Expected %s, got %s", expected, merged)
	}
}