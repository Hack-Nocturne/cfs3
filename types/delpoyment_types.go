package types

import "time"

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

type Stage struct {
	Name      string     `json:"name"`
	StartedOn *time.Time `json:"started_on"`
	EndedOn   *time.Time `json:"ended_on"`
	Status    string     `json:"status"`
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

type DeploymentConfig struct {
	FailOpen                         bool     `json:"fail_open"`
	AlwaysUseLatestCompatibilityDate bool     `json:"always_use_latest_compatibility_date"`
	CompatibilityDate                string   `json:"compatibility_date"`
	CompatibilityFlags               []string `json:"compatibility_flags"`
	BuildImageMajorVersion           int      `json:"build_image_major_version"`
	UsageModel                       string   `json:"usage_model"`
}

type DeploymentConfigs struct {
	Preview    DeploymentConfig `json:"preview"`
	Production DeploymentConfig `json:"production"`
}

// PagesDeployOptions holds the minimal options required for a Pages deployment.
type PagesDeployOptions struct {
	Directory   string // path to static assets
	AccountId   string // Cloudflare account ID
	ProjectName string // Cloudflare Pages project name
	Branch      string // branch name (if empty, assumed production)
	SkipCaching bool   // whether to skip caching
}
