package worker

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Hack-Nocturne/cfs3/vars"
)

// FilePatch represents a single patch operation.
type FilePatch struct {
	LocalFile string         `json:"local_file"`
	Remote    string         `json:"remote_dir"`
	Metadata  map[string]any `json:"metadata"`
}

// Config represents the top-level configuration.
type Config struct {
	By          string            `json:"by"`
	Mode        string            `json:"mode"`
	ProjectName string            `json:"project_name"`
	Headers     map[string]string `json:"headers,omitempty"`
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

	fileMap, err := cfg.processPatchFiles(vars.UPLOAD_BASE_DIR)
	if err != nil {
		return nil, fmt.Errorf("error processing patch files: %w", err)
	}

	if err := cfg.createHeadersFile(vars.UPLOAD_BASE_DIR, fileMap); err != nil {
		return nil, fmt.Errorf("error creating headers file: %w", err)
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

		if len(c.FilesPatch) > 40 {
			return errors.New("mode 'patch' supports a maximum of 40 files in files__patch in one run")
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

	// Add default headers
	c.Headers["x-powered-by"] = "CFS3"
	c.Headers["x-developed-by"] = "Rishabh Kumar"
	c.Headers["x-contact-email"] = "rishabh.kumar.pro@gmail.com"

	if c.Headers != nil {
		if len(c.Headers)-3 > 40 { // subtracting 3 for the default headers we add
			return errors.New("headers must contain at most 40 entries")
		}

		for k, v := range c.Headers {
			k = strings.ToLower(strings.TrimSpace(k))
			v = strings.TrimSpace(v)
			c.Headers[k] = v

			if k == "" || v == "" {
				return errors.New("header keys and values must be non-empty strings")
			}

			// Stay within Cloudflare Pages limits header size limits (2k per line)
			if len(fmt.Sprintf("%s: %s", k, v)) > 1800 {
				return fmt.Errorf("header %q exceeds 1800 character limit", k)
			}
		}
	}

	for i, fp := range c.FilesPatch {
		if fp.LocalFile == "" {
			return fmt.Errorf("files__patch[%d]: field 'local_file' is required", i)
		}
		if fp.Remote == "" {
			return fmt.Errorf("files__patch[%d]: field 'remote_dir' is required", i)
		}

		fileName := path.Base(fp.LocalFile)
		if len(fileName) > 900 {
			return fmt.Errorf("files__patch[%d]: local file name exceeds 900 character limit", i)
		}
	}

	return nil
}

// processPatchFiles processes the patch files, generating SHA1 hashes and copying them to the specified directory.
// It also ensures the directory structure is created as needed.
func (c *Config) processPatchFiles(parentDir string) (map[string]string, error) {
	if c.Mode != "patch" {
		return nil, nil // No-op for non-patch mode
	}

	fileNameMap := make(map[string]string)
	for i, fp := range c.FilesPatch {
		f, err := os.Open(fp.LocalFile)
		if err != nil {
			return nil, fmt.Errorf("opening %q: %w", fp.LocalFile, err)
		}
		defer f.Close()

		hasher := sha1.New()
		if _, err := io.Copy(hasher, f); err != nil {
			return nil, fmt.Errorf("hashing %q: %w", fp.LocalFile, err)
		}
		sha1hex := hex.EncodeToString(hasher.Sum(nil))
		ext := strings.TrimPrefix(filepath.Ext(fp.LocalFile), ".")

		fp.Remote = path.Clean(fp.Remote)
		fp.Remote = strings.TrimPrefix(fp.Remote, "/")
		fp.Remote = path.Join(fp.Remote, fmt.Sprintf("%s.%s", sha1hex, ext))
		fp.Remote = filepath.ToSlash(fp.Remote)

		fileNameMap[fp.Remote] = filepath.Base(fp.LocalFile)

		dest := path.Join(parentDir, fp.Remote)
		if err := os.MkdirAll(path.Dir(dest), 0o755); err != nil {
			return nil, fmt.Errorf("making dirs for %q: %w", dest, err)
		}

		// Copy file content a second time from start
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return nil, fmt.Errorf("rewinding %q: %w", fp.LocalFile, err)
		}
		out, err := os.Create(dest)
		if err != nil {
			return nil, fmt.Errorf("creating %q: %w", dest, err)
		}
		if _, err := io.Copy(out, f); err != nil {
			out.Close()
			return nil, fmt.Errorf("copying to %q: %w", dest, err)
		}
		out.Close()

		c.FilesPatch[i] = fp
	}

	return fileNameMap, nil
}

// createHeadersFile writes a Cloudflare Pages compatible "_headers" file
// at the root of parentDir. It emits a global rule ("/*") with all headers.
// If no headers are configured, this is a no-op.
func (c *Config) createHeadersFile(parentDir string, fileMap map[string]string) error {
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
	keys := make([]string, 0, len(c.Headers))
	for k := range c.Headers {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// write each header line
	for _, k := range keys {
		fmt.Fprintf(f, "  %s: %s\n", k, c.Headers[k])
	}

	// write content-disposition lines for each file
	for remote, name := range fileMap {
		fmt.Fprintf(f,
			"/%s\n  content-disposition: attachment; filename=\"%s\"; filename*=UTF-8''%s\n",
			remote,
			name,
			url.PathEscape(name),
		)
	}

	return nil
}
