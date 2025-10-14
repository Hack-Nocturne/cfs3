package cfs3

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"

	"github.com/Hack-Nocturne/cfs3/types"
	"github.com/Hack-Nocturne/cfs3/utils"
	"github.com/Hack-Nocturne/cfs3/vars"
	"github.com/Hack-Nocturne/cfs3/worker"
)

// FilePatch represents a single patch operation.
type FilePatch struct {
	LocalFile string         `json:"local_file"`
	Remote    string         `json:"remote_dir"`
	Metadata  map[string]any `json:"metadata"`
}

// CFS3Config represents the top-level configuration.
type CFS3Config struct {
	By          string            `json:"by"`
	Mode        string            `json:"mode"`
	ProjectName string            `json:"project_name"`
	Headers     map[string]string `json:"headers,omitempty"`
	FilesPatch  []FilePatch       `json:"files__patch,omitempty"`
	FilesRemove []int64           `json:"files__remove,omitempty"`

	isProcessed bool
	metadata    map[string]types.FileContainer
}

// NewCFS3ConfigFromFile reads a JSON file, unmarshals into struct and creates cfs3 config instance.
func NewCFS3ConfigFromFile(path string) (*CFS3Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg CFS3Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config JSON: %w", err)
	}

	return &cfg, nil
}

func (c *CFS3Config) Process() error {
	if c.isProcessed {
		return nil
	}
	c.isProcessed = true

	if err := c.validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	fileMap, err := c.processPatchFiles(vars.UPLOAD_BASE_DIR)
	if err != nil {
		return fmt.Errorf("error processing patch files: %w", err)
	}

	if err := c.createHeadersFile(vars.UPLOAD_BASE_DIR, fileMap); err != nil {
		return fmt.Errorf("error creating headers file: %w", err)
	}

	// Add default headers
	c.Headers["x-powered-by"] = "CFS3"
	c.Headers["x-developed-by"] = "Rishabh Kumar"
	c.Headers["x-contact-email"] = "rishabh.kumar.pro@gmail.com"

	// The trick here is to include existing files metadata used by Cloudflare, then
	// a) For patch mode, we add new files to existing metadata
	// b) For remove mode, we exclude the removed files from existing metadata
	// This way we always deploy the project with full metadata-set required without uploading same files again

	if c.Mode == "remove" {
		meta, meErr := worker.FetchAllMetaExcluding(c.ProjectName, c.FilesRemove)
		if meErr != nil {
			return fmt.Errorf("failure fetching existing meta: %v", meErr)
		}

		c.metadata = meta
	} else {
		meta, maErr := worker.FetchAllMeta(c.ProjectName)
		if maErr != nil {
			return fmt.Errorf("failure fetching existing meta: %v", maErr)
		}

		c.metadata = meta
	}

	return nil
}

func (c *CFS3Config) Apply() error {
	if !c.isProcessed {
		return fmt.Errorf("use Process() method before Apply()")
	}

	os.MkdirAll(vars.UPLOAD_BASE_DIR, 0o755)
	defer func() { os.RemoveAll(vars.UPLOAD_BASE_DIR) }()

	uploadArgs := types.PagesDeployOptions{
		Directory:   vars.UPLOAD_BASE_DIR,
		AccountId:   vars.CF_ACCOUNT_ID,
		ProjectName: c.ProjectName,
		SkipCaching: false,
	}

	deployResp, fileMap, err := utils.Deploy(uploadArgs, c.Mode == "patch")
	if err != nil {
		fmt.Println("❌ Deployment failed: " + err.Error())
		return err
	}

	fmt.Println("💫 Deployment completed with ID: " + deployResp.ID)
	fmt.Println("🌐 Take a peek over " + deployResp.URL)
	maps.Copy(c.metadata, fileMap)

	return c.upsertMetadata()
}
