package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/Hack-Nocturne/cfs3/types"
)

// Deploy publishes the directory to Cloudflare Pages by performing the following steps:
//  1. Reads optional configuration files (_headers, _redirects, _routes.json, _worker.js)
//  2. Fetches project info from Cloudflare.
//  3. Validates the directory and uploads static assets to generate a manifest.
//  4. Constructs a multipart payload including the manifest and worker bundle.
//  5. Sends a POST request to the deployment endpoint with retry logic.
func Deploy(options types.PagesDeployOptions, isPatchMode bool) (*types.DeploymentResponse, map[string]types.FileContainer, error) {
	directory := options.Directory
	accountId := options.AccountId
	projectName := options.ProjectName
	branch := options.Branch
	skipCaching := options.SkipCaching

	// Read optional files from the directory.
	var headersContent, redirectsContent, routesCustomContent, workerJSContent string

	// Read _headers (if exists).
	headersPath := filepath.Join(directory, "_headers")
	if data, err := os.ReadFile(headersPath); err == nil {
		headersContent = string(data)
	}

	// Read _redirects (if exists).
	redirectsPath := filepath.Join(directory, "_redirects")
	if data, err := os.ReadFile(redirectsPath); err == nil {
		redirectsContent = string(data)
	}

	// Read _routes.json (if exists).
	routesPath := filepath.Join(directory, "_routes.json")
	if data, err := os.ReadFile(routesPath); err == nil {
		routesCustomContent = string(data)
	}

	// Process _worker.js: if it is a directory, try reading an entry file (e.g. index.js),
	// otherwise read the file content.
	workerPath := filepath.Join(directory, "_worker.js")
	if fi, err := os.Stat(workerPath); err == nil {
		if fi.IsDir() {
			indexPath := filepath.Join(workerPath, "index.js")
			if data, err := os.ReadFile(indexPath); err == nil {
				workerJSContent = string(data)
			}
		} else {
			if data, err := os.ReadFile(workerPath); err == nil {
				workerJSContent = string(data)
			}
		}
	}

	// Fetch project info from Cloudflare.
	projectUrl := fmt.Sprintf("/accounts/%s/pages/projects/%s", accountId, projectName)
	if _, err := fetchResult[types.ProjectResponse](projectUrl, "GET", nil, nil); err != nil {
		return nil, nil, fmt.Errorf("failed to fetch project info: %v", err)
	}

	// Validate the directory and get a file map.
	fileMap, err := validate(directory, isPatchMode)
	if err != nil {
		return nil, nil, fmt.Errorf("validation error: %v", err)
	}

	// Upload static assets and obtain the manifest.
	uploadArgs := types.UploadArgs{
		FileMap:     fileMap,
		AccountId:   accountId,
		ProjectName: projectName,
		SkipCaching: skipCaching,
	}
	manifest, err := upload(uploadArgs)
	if err != nil {
		return nil, nil, fmt.Errorf("upload error: %v", err)
	}

	// Build a multipart form-data payload.
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add the manifest (as JSON).
	manifestJson, err := json.Marshal(manifest)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal manifest: %v", err)
	}
	if err = writer.WriteField("manifest", string(manifestJson)); err != nil {
		return nil, nil, fmt.Errorf("failed to write manifest field: %v", err)
	}

	// Include branch information if provided.
	if branch != "" {
		if err = writer.WriteField("branch", branch); err != nil {
			return nil, nil, fmt.Errorf("failed to write branch field: %v", err)
		}
	}

	// Append _headers file if available.
	if headersContent != "" {
		part, err := writer.CreateFormFile("_headers", "_headers")
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create _headers form file: %v", err)
		}
		part.Write([]byte(headersContent))
	}

	// Append _redirects file if available.
	if redirectsContent != "" {
		part, err := writer.CreateFormFile("_redirects", "_redirects")
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create _redirects form file: %v", err)
		}
		part.Write([]byte(redirectsContent))
	}

	// Append _routes.json file if available.
	if routesCustomContent != "" {
		part, err := writer.CreateFormFile("_routes.json", "_routes.json")
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create _routes.json form file: %v", err)
		}
		part.Write([]byte(routesCustomContent))
	}

	// Append the worker bundle if _worker.js content was found.
	if workerJSContent != "" {
		// In this simplified example the bundle is just the content from _worker.js (or its entry file).
		part, err := writer.CreateFormFile("_worker.bundle", "_worker.bundle")
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create _worker.bundle form file: %v", err)
		}
		part.Write([]byte(workerJSContent))
	}

	// Finalize the form-data payload.
	if err = writer.Close(); err != nil {
		return nil, nil, fmt.Errorf("failed to close form writer: %v", err)
	}

	// Prepare to POST the deployment.
	deployURL := fmt.Sprintf("/accounts/%s/pages/projects/%s/deployments", accountId, projectName)
	headers := map[string]string{
		"Content-Type": writer.FormDataContentType(),
	}

	maxAttempts := 3
	var lastErr error
	// Retry loop with exponential backoff.
	for attempts := range maxAttempts {
		if deploymentResponse, err := fetchResult[types.DeploymentResponse](deployURL, "POST", headers, buf.Bytes()); err == nil {
			return &deploymentResponse.Result, fileMap, nil
		}
		lastErr = err
		time.Sleep(time.Duration(1<<attempts) * time.Second)
	}

	return nil, nil, fmt.Errorf("deployment failed after %d attempts: %v", maxAttempts, lastErr)
}
