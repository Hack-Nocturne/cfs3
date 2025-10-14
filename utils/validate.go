package utils

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Hack-Nocturne/cfs3/types"
	"github.com/Hack-Nocturne/cfs3/vars"
	"github.com/bmatcuk/doublestar/v4"
	"github.com/zeebo/blake3"
)

// Ignore patterns (similar to the Node.js ignore list)
var ignorePatterns = []string{
	"_worker.js",
	"_redirects",
	"_headers",
	"_routes.json",
	"functions",
	"**/.DS_Store",
	"**/node_modules",
	"**/.git",
}

// shouldIgnore checks if a given relative path matches one of the ignore patterns.
func shouldIgnore(relPath string) bool {
	// Ensure a consistent (unix-style) path separator.
	relPath = filepath.ToSlash(relPath)
	for _, pattern := range ignorePatterns {
		// doublestar supports patterns like "**/node_modules"
		if match, err := doublestar.PathMatch(pattern, relPath); err == nil && match {
			return true
		}
	}
	return false
}

// fileTask represents a file to be processed.
type fileTask struct {
	relative  string
	fullPath  string
	size      int64
	extension string
}

// validate walks the directory, processes files concurrently,
// and returns a map of relative paths to FileContainer.
func validate(directory string, isPatchMode bool) (map[string]types.FileContainer, error) {
	absDir, err := filepath.Abs(directory)
	if err != nil {
		return nil, err
	}

	if !isPatchMode {
		return nil, nil
	}

	startTime := time.Now()
	var fileMap map[string]types.FileContainer

	defer func() {
		duration := time.Since(startTime).Seconds()
		fmt.Printf("Validation took %.2f seconds\n", duration)
	}()

	var tasks []fileTask

	// Walk the directory tree.
	err = filepath.WalkDir(absDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(absDir, path)
		if err != nil {
			return err
		}
		relPath = filepath.ToSlash(relPath)

		// Skip ignored files/directories.
		if shouldIgnore(relPath) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip symbolic links.
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}

		if d.IsDir() {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		// Check file size.
		if info.Size() > vars.MAX_ASSET_SIZE {
			return fmt.Errorf("file %s is %d bytes, exceeds maximum allowed %d bytes", relPath, info.Size(), vars.MAX_ASSET_SIZE)
		}

		ext := strings.TrimPrefix(filepath.Ext(path), ".")
		tasks = append(tasks, fileTask{
			relative:  relPath,
			fullPath:  path,
			size:      info.Size(),
			extension: ext,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Check overall file count.
	if len(tasks) > vars.MAX_ASSET_COUNT {
		return nil, fmt.Errorf("number of files %d exceeds maximum allowed %d", len(tasks), vars.MAX_ASSET_COUNT)
	}

	var mu sync.Mutex
	var wg sync.WaitGroup

	workerCount := runtime.NumCPU()
	tasksChan := make(chan fileTask, len(tasks))

	// Start worker goroutines.
	for range workerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for task := range tasksChan {
				// Read file contents.
				data, err := os.ReadFile(task.fullPath)
				if err != nil {
					fmt.Printf("Error reading file %s: %v", task.fullPath, err)
					continue
				}
				// Encode the file's contents as base64.
				base64Content := base64.StdEncoding.EncodeToString(data)
				// Concatenate the base64 content with the file extension.
				input := base64Content + task.extension

				// Compute Blake3 hash.
				hashBytes := blake3.Sum256([]byte(input))
				hexStr := hex.EncodeToString(hashBytes[:])
				hashFinal := hexStr[:32] // take the first 32 hex characters

				// Determine the MIME type based on the file extension.
				extWithDot := filepath.Ext(task.relative)
				mimeType := ExtToMimeType(extWithDot)
				if mimeType == "" {
					mimeType = "application/octet-stream"
				}

				container := types.FileContainer{
					Path:        task.fullPath,
					ContentType: mimeType,
					SizeInBytes: task.size,
					Hash:        hashFinal,
				}

				// Protect concurrent map writes.
				mu.Lock()
				fileMap[task.relative] = container
				mu.Unlock()
			}
		}()
	}

	// Send file tasks to the workers.
	for _, task := range tasks {
		tasksChan <- task
	}
	close(tasksChan)
	wg.Wait()

	return fileMap, nil
}
