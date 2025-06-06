package download

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/schollz/progressbar/v3"
)

// ChunkDownloader handles parallel chunk downloads
type ChunkDownloader struct {
	URL         string
	Filename    string
	ChunkSize   int64
	Parallelism int
}

// Chunk represents a download chunk
type Chunk struct {
	Start int64
	End   int64
	Index int
}

// NewChunkDownloader creates a new chunk downloader
func NewChunkDownloader(url, filename string) *ChunkDownloader {
	return &ChunkDownloader{
		URL:         url,
		Filename:    filename,
		ChunkSize:   1024 * 1024, // 1MB chunks
		Parallelism: 4,           // 4 parallel downloads
	}
}

// Download downloads a file with parallel chunks
func (cd *ChunkDownloader) Download(ctx context.Context) error {
	// Get file size
	resp, err := http.Head(cd.URL)
	if err != nil {
		return fmt.Errorf("failed to get file info: %w", err)
	}
	resp.Body.Close()

	// Check if server supports range requests
	if resp.Header.Get("Accept-Ranges") != "bytes" {
		// Fallback to single download
		return cd.downloadSingle(ctx)
	}

	contentLength := resp.ContentLength
	if contentLength <= 0 {
		return cd.downloadSingle(ctx)
	}

	// Create chunks
	chunks := cd.createChunks(contentLength)
	
	// Create progress bar
	bar := progressbar.DefaultBytes(
		contentLength,
		fmt.Sprintf("Downloading %s", filepath.Base(cd.Filename)),
	)

	// Create temporary files for each chunk
	tempFiles := make([]*os.File, len(chunks))
	defer func() {
		for _, f := range tempFiles {
			if f != nil {
				f.Close()
				os.Remove(f.Name())
			}
		}
	}()

	// Download chunks in parallel
	var wg sync.WaitGroup
	errChan := make(chan error, len(chunks))

	for i, chunk := range chunks {
		wg.Add(1)
		go func(idx int, c Chunk) {
			defer wg.Done()
			
			tempFile, err := os.CreateTemp("", fmt.Sprintf("chunk_%d_*", idx))
			if err != nil {
				errChan <- err
				return
			}
			tempFiles[idx] = tempFile

			if err := cd.downloadChunk(ctx, c, tempFile, bar); err != nil {
				errChan <- err
			}
		}(i, chunk)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	if err := <-errChan; err != nil {
		return fmt.Errorf("chunk download failed: %w", err)
	}

	// Merge chunks
	return cd.mergeChunks(tempFiles)
}

// createChunks creates download chunks
func (cd *ChunkDownloader) createChunks(contentLength int64) []Chunk {
	var chunks []Chunk
	
	for i := int64(0); i < contentLength; i += cd.ChunkSize {
		end := i + cd.ChunkSize - 1
		if end >= contentLength {
			end = contentLength - 1
		}
		
		chunks = append(chunks, Chunk{
			Start: i,
			End:   end,
			Index: len(chunks),
		})
	}
	
	return chunks
}

// downloadChunk downloads a single chunk
func (cd *ChunkDownloader) downloadChunk(ctx context.Context, chunk Chunk, file *os.File, bar *progressbar.ProgressBar) error {
	req, err := http.NewRequestWithContext(ctx, "GET", cd.URL, nil)
	if err != nil {
		return err
	}

	// Set range header
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", chunk.Start, chunk.End))

	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("server doesn't support range requests: %d", resp.StatusCode)
	}

	// Copy with progress
	_, err = io.Copy(io.MultiWriter(file, bar), resp.Body)
	return err
}

// downloadSingle downloads file in a single request (fallback)
func (cd *ChunkDownloader) downloadSingle(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, "GET", cd.URL, nil)
	if err != nil {
		return err
	}

	client := &http.Client{
		Timeout: 10 * time.Minute,
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: %d", resp.StatusCode)
	}

	// Create output file
	out, err := os.Create(cd.Filename)
	if err != nil {
		return err
	}
	defer out.Close()

	// Create progress bar
	var bar *progressbar.ProgressBar
	if resp.ContentLength > 0 {
		bar = progressbar.DefaultBytes(
			resp.ContentLength,
			fmt.Sprintf("Downloading %s", filepath.Base(cd.Filename)),
		)
	} else {
		bar = progressbar.DefaultBytes(
			-1,
			fmt.Sprintf("Downloading %s", filepath.Base(cd.Filename)),
		)
	}

	// Copy with progress
	_, err = io.Copy(io.MultiWriter(out, bar), resp.Body)
	return err
}

// mergeChunks merges temporary chunk files into final file
func (cd *ChunkDownloader) mergeChunks(tempFiles []*os.File) error {
	// Create output file
	out, err := os.Create(cd.Filename)
	if err != nil {
		return err
	}
	defer out.Close()

	// Merge chunks in order
	for _, tempFile := range tempFiles {
		if tempFile == nil {
			continue
		}
		
		// Seek to beginning of temp file
		if _, err := tempFile.Seek(0, 0); err != nil {
			return err
		}

		// Copy chunk to output file
		if _, err := io.Copy(out, tempFile); err != nil {
			return err
		}
	}

	return nil
}