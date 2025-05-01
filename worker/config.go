package worker

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Hack-Nocturne/cfs3/vars"
)

// FilePatch represents a single patch operation.
type FilePatch struct {
	Local    string         `json:"local"`
	Remote   string         `json:"remote"`
	Metadata map[string]any `json:"metadata"`
}

// Config represents the top-level configuration.
type Config struct {
	By          string            `json:"by"`
	Mode        string            `json:"mode"`
	ProjectName string            `json:"project_name"`
	Header      map[string]string `json:"header,omitempty"`
	FilesPatch  []FilePatch       `json:"files__patch,omitempty"`
	FilesRemove []int64           `json:"files__remove,omitempty"`
}

// LoadConfig reads a JSON file, unmarshals into Config and validates it.
func LoadNProcessConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config JSON: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	if err := cfg.createHeadersFile(vars.UPLOAD_BASE_DIR); err != nil {
		return nil, fmt.Errorf("error creating headers file: %w", err)
	}

	if err := cfg.processPatchFiles(vars.UPLOAD_BASE_DIR); err != nil {
		return nil, fmt.Errorf("error processing patch files: %w", err)
	}

	return &cfg, nil
}

// validate enforces required fields and mode-specific constraints.
func (c *Config) validate() error {
	if c.By == "" {
		return errors.New("field 'by' is required")
	}
	if c.ProjectName == "" {
		return errors.New("field 'project_name' is required")
	}
	switch c.Mode {
	case "patch":
		if len(c.FilesPatch) == 0 {
			return errors.New("mode 'patch' requires non-empty files__patch")
		}
	case "remove":
		if len(c.FilesRemove) == 0 {
			return errors.New("mode 'remove' requires non-empty files__remove")
		}
	case "list":
		return errors.New("mode 'list' currently not implemented")
	default:
		return errors.New("mode unknown")
	}
	if c.Header != nil {
		for k, v := range c.Header {
			c.Header[strings.ToLower(strings.TrimSpace(k))] = strings.TrimSpace(v)
			if strings.TrimSpace(k) == "" || strings.TrimSpace(v) == "" {
				return errors.New("header keys and values must be non-empty strings")
			}
		}
	}
	return nil
}

// processPatchFiles processes the patch files, generating SHA1 hashes and copying them to the specified directory.
// It also ensures the directory structure is created as needed.
func (c *Config) processPatchFiles(parentDir string) error {
	if c.Mode != "patch" {
		return nil // No-op for non-patch mode
	}

	for i, fp := range c.FilesPatch {
		f, err := os.Open(fp.Local)
		if err != nil {
			return fmt.Errorf("opening %q: %w", fp.Local, err)
		}
		defer f.Close()

		hasher := sha1.New()
		if _, err := io.Copy(hasher, f); err != nil {
			return fmt.Errorf("hashing %q: %w", fp.Local, err)
		}
		sha1hex := hex.EncodeToString(hasher.Sum(nil))
		ext := strings.TrimPrefix(filepath.Ext(fp.Local), ".")

		fp.Remote = path.Clean(fp.Remote)
		fp.Remote = strings.TrimPrefix(fp.Remote, "/")
		fp.Remote = path.Join(fp.Remote, fmt.Sprintf("%s.%s", sha1hex, ext))
		fp.Remote = filepath.ToSlash(fp.Remote)

		dest := path.Join(parentDir, fp.Remote)
		if err := os.MkdirAll(path.Dir(dest), 0o755); err != nil {
			return fmt.Errorf("making dirs for %q: %w", dest, err)
		}

		// Copy file content a second time from start
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return fmt.Errorf("rewinding %q: %w", fp.Local, err)
		}
		out, err := os.Create(dest)
		if err != nil {
			return fmt.Errorf("creating %q: %w", dest, err)
		}
		if _, err := io.Copy(out, f); err != nil {
			out.Close()
			return fmt.Errorf("copying to %q: %w", dest, err)
		}
		out.Close()

		c.FilesPatch[i] = fp
	}
	return nil
}

// createHeadersFile writes a Cloudflare Pages compatible "_headers" file
// at the root of parentDir. It emits a global rule ("/*") with all headers.
// If no headers are configured, this is a no-op.
func (c *Config) createHeadersFile(parentDir string) error {
	if len(c.Header) == 0 {
		return nil
	}

	// ensure the target directory exists
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return fmt.Errorf("making dirs for headers file: %w", err)
	}

	outPath := filepath.Join(parentDir, "_headers")
	f, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("creating %q: %w", outPath, err)
	}
	defer f.Close()

	fmt.Fprintln(f, "/*")

	// sort header keys for stable output
	keys := make([]string, 0, len(c.Header))
	for k := range c.Header {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// write each header line
	for _, k := range keys {
		fmt.Fprintf(f, "  %s: %s\n", k, c.Header[k])
	}

	return nil
}
