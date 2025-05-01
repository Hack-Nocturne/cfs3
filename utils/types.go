package utils

import (
	"fmt"
	"time"

	"github.com/Hack-Nocturne/cfs3/types"
)

// UploadPayloadFile represents a file upload payload.
type UploadPayloadFile struct {
	Key      string `json:"key"`
	Value    string `json:"value"`
	Metadata struct {
		ContentType string `json:"contentType"`
	} `json:"metadata"`
	Base64 bool `json:"base64"`
}

// UploadArgs holds parameters for the upload function.
type UploadArgs struct {
	FileMap     map[string]types.FileContainer
	Jwt         *string
	AccountId   string
	ProjectName string
	SkipCaching bool
}

// APIError represents an error response from the API.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("APIError: %d: %s", e.StatusCode, e.Message)
}

// CFResponse represents a response from Cloudflare API.
type CFResponse[T any] struct {
	Result   T        `json:"result"`
	Success  bool     `json:"success"`
	Errors   []string `json:"errors"`
	Messages []string `json:"messages"`
}

// Represents the response from the upload API.
type UploadResponse struct {
	SuccessfullKeyCount int      `json:"successful_key_count"`
	UnsuccessfulKeys    []string `json:"unsuccessful_keys"`
}

// Represents the deployment response from Cloudflare, after a successful Pages deployment.
type DeploymentResponse struct {
	ID                     string            `json:"id"`
	ShortID                string            `json:"short_id"`
	ProjectID              string            `json:"project_id"`
	ProjectName            string            `json:"project_name"`
	Environment            string            `json:"environment"`
	URL                    string            `json:"url"`
	CreatedOn              time.Time         `json:"created_on"`
	ModifiedOn             time.Time         `json:"modified_on"`
	LatestStage            Stage             `json:"latest_stage"`
	DeploymentTrigger      DeploymentTrigger `json:"deployment_trigger"`
	Stages                 []Stage           `json:"stages"`
	BuildConfig            BuildConfig       `json:"build_config"`
	EnvVars                map[string]string `json:"env_vars"`
	CompatibilityDate      string            `json:"compatibility_date"`
	CompatibilityFlags     []string          `json:"compatibility_flags"`
	BuildImageMajorVersion int               `json:"build_image_major_version"`
	UsageModel             *string           `json:"usage_model"`
	Aliases                *[]string         `json:"aliases"`
	IsSkipped              bool              `json:"is_skipped"`
	ProductionBranch       string            `json:"production_branch"`
}

type Stage struct {
	Name      string     `json:"name"`
	StartedOn *time.Time `json:"started_on"`
	EndedOn   *time.Time `json:"ended_on"`
	Status    string     `json:"status"`
}

type DeploymentTriggerMetadata struct {
	Branch        string `json:"branch"`
	CommitHash    string `json:"commit_hash"`
	CommitMessage string `json:"commit_message"`
	CommitDirty   bool   `json:"commit_dirty"`
}

type DeploymentTrigger struct {
	Type     string                    `json:"type"`
	Metadata DeploymentTriggerMetadata `json:"metadata"`
}

type BuildConfig struct {
	BuildCommand      *string `json:"build_command"`
	DestinationDir    *string `json:"destination_dir"`
	BuildCaching      *bool   `json:"build_caching"`
	RootDir           *string `json:"root_dir"`
	WebAnalyticsTag   *string `json:"web_analytics_tag"`
	WebAnalyticsToken *string `json:"web_analytics_token"`
}

// Represents the project details and it's metadata from Cloudflare Pages.
type ProjectResponse struct {
	Id                   string            `json:"id"`
	Name                 string            `json:"name"`
	Subdomain            string            `json:"subdomain"`
	Domains              []string          `json:"domains"`
	ProductionBranch     string            `json:"production_branch"`
	CreatedOn            string            `json:"created_on"`
	ProductionScriptName string            `json:"production_script_name"`
	PreviewScriptName    string            `json:"preview_script_name"`
	DeploymentConfigs    DeploymentConfigs `json:"deployment_configs"`
}

type DeploymentConfigs struct {
	Preview    DeploymentConfig `json:"preview"`
	Production DeploymentConfig `json:"production"`
}

type DeploymentConfig struct {
	FailOpen                         bool     `json:"fail_open"`
	AlwaysUseLatestCompatibilityDate bool     `json:"always_use_latest_compatibility_date"`
	CompatibilityDate                string   `json:"compatibility_date"`
	CompatibilityFlags               []string `json:"compatibility_flags"`
	BuildImageMajorVersion           int      `json:"build_image_major_version"`
	UsageModel                       string   `json:"usage_model"`
}

// PagesDeployOptions holds the minimal options required for a Pages deployment.
type PagesDeployOptions struct {
	Directory   string // path to static assets
	AccountId   string // Cloudflare account ID
	ProjectName string // Cloudflare Pages project name
	Branch      string // branch name (if empty, assumed production)
	SkipCaching bool   // whether to skip caching
}
