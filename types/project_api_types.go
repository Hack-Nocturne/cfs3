package types

import (
	"fmt"
)

type BuildConfig struct {
	BuildCommand      *string `json:"build_command"`
	DestinationDir    *string `json:"destination_dir"`
	BuildCaching      *bool   `json:"build_caching"`
	RootDir           *string `json:"root_dir"`
	WebAnalyticsTag   *string `json:"web_analytics_tag"`
	WebAnalyticsToken *string `json:"web_analytics_token"`
}

// APIError represents an error response from the API.
type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("APIError: %d: %s", e.StatusCode, e.Message)
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

// CFResponse represents a response from Cloudflare API.
type CFResponse[T any] struct {
	Result   T        `json:"result"`
	Success  bool     `json:"success"`
	Errors   []string `json:"errors"`
	Messages []string `json:"messages"`
}
