package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"sync"
	"time"

	"github.com/Hack-Nocturne/cfs3/types"
	"github.com/Hack-Nocturne/cfs3/utils"
	"github.com/Hack-Nocturne/cfs3/vars"
)

// upload processes file uploads by first determining missing file hashes,
// bucketing files, and concurrently uploading each bucket.
func Upload(args utils.UploadArgs) (map[string]string, error) {
	// fetchJwt returns a JWT string either from the provided args
	// or by calling the API endpoint.
	fetchJwt := func() (string, error) {
		if args.Jwt != nil && *args.Jwt != "" {
			return *args.Jwt, nil
		}

		type JwtResponse struct {
			JWT string `json:"jwt"`
		}
		jwtResp, err := utils.FetchResult[JwtResponse](
			fmt.Sprintf("/accounts/%s/pages/projects/%s/upload-token", args.AccountId, args.ProjectName),
			"GET",
			nil,
			nil,
		)
		if err != nil {
			return "", err
		}

		return jwtResp.Result.JWT, nil
	}

	// Convert the file map to a slice.
	var files []types.FileContainer
	for _, f := range args.FileMap {
		files = append(files, f)
	}

	jwt, err := fetchJwt()
	if err != nil {
		return nil, err
	}

	start := time.Now()
	attempts := 0

	// getMissingHashes fetches the list of missing file hashes.
	var getMissingHashes func(skipCaching bool) ([]string, error)
	getMissingHashes = func(skipCaching bool) ([]string, error) {
		if skipCaching {
			hashes := make([]string, len(files))
			for i, file := range files {
				hashes[i] = file.Hash
			}
			return hashes, nil
		}

		payloadData := map[string][]string{
			"hashes": {},
		}
		for _, file := range files {
			payloadData["hashes"] = append(payloadData["hashes"], file.Hash)
		}
		payloadBytes, err := json.Marshal(payloadData)
		if err != nil {
			return nil, err
		}
		headers := map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + jwt,
		}

		missingResp, err := utils.FetchResult[[]string]("/pages/assets/check-missing", "POST", headers, payloadBytes)
		if err != nil {
			if attempts < vars.MAX_CHECK_MISSING_ATTEMPTS {
				time.Sleep(time.Second * time.Duration(1<<attempts))
				attempts++
				// If unauthorized or JWT expired, refresh the token.
				if apiErr, ok := err.(*utils.APIError); ok && apiErr.StatusCode == 401 {
					if newJwt, err := fetchJwt(); err == nil {
						jwt = newJwt
					}
				} else if expired, _ := utils.IsJwtExpired(jwt); expired {
					if newJwt, err := fetchJwt(); err == nil {
						jwt = newJwt
					}
				}
				return getMissingHashes(skipCaching)
			}
			return nil, err
		}
		return missingResp.Result, nil
	}

	missingHashes, err := getMissingHashes(args.SkipCaching)
	if err != nil {
		return nil, err
	}

	// Filter files that need to be uploaded.
	var sortedFiles []types.FileContainer
	missingSet := make(map[string]bool)
	for _, h := range missingHashes {
		missingSet[h] = true
	}
	for _, file := range files {
		if missingSet[file.Hash] {
			sortedFiles = append(sortedFiles, file)
		}
	}
	// Sort descending by file size.
	sort.Slice(sortedFiles, func(i, j int) bool {
		return sortedFiles[i].SizeInBytes > sortedFiles[j].SizeInBytes
	})

	// Bucket the files.
	type Bucket struct {
		Files         []types.FileContainer
		RemainingSize int64
	}
	var buckets []Bucket
	// Start with a few buckets so that even small projects benefit from concurrency.
	for range vars.BULK_UPLOAD_CONCURRENCY {
		buckets = append(buckets, Bucket{
			Files:         []types.FileContainer{},
			RemainingSize: vars.MAX_BUCKET_SIZE,
		})
	}
	bucketOffset := 0
	for _, file := range sortedFiles {
		inserted := false
		for i := range buckets {
			idx := (i + bucketOffset) % len(buckets)
			bucket := &buckets[idx]
			if bucket.RemainingSize >= file.SizeInBytes && len(bucket.Files) < vars.MAX_BUCKET_FILE_COUNT {
				bucket.Files = append(bucket.Files, file)
				bucket.RemainingSize -= file.SizeInBytes
				inserted = true
				break
			}
		}
		if !inserted {
			buckets = append(buckets, Bucket{
				Files:         []types.FileContainer{file},
				RemainingSize: vars.MAX_BUCKET_SIZE - file.SizeInBytes,
			})
		}
		bucketOffset++
	}

	// Set up progress reporting.
	counter := len(args.FileMap) - len(sortedFiles)
	utils.IncUpTo(args.ProjectName, counter, len(args.FileMap))

	// Use a semaphore to limit concurrency.
	sem := make(chan struct{}, vars.BULK_UPLOAD_CONCURRENCY)
	var wg sync.WaitGroup
	var uploadErr error
	var mu sync.Mutex

	// For each bucket, run an upload goroutine.
	for _, bucket := range buckets {
		if len(bucket.Files) == 0 {
			continue
		}
		wg.Add(1)
		go func(bucket Bucket) {
			defer wg.Done()
			attempts := 0
			gatewayErrors := 0

			var doUpload func() error
			doUpload = func() error {
				// Build the payload.
				payload := make([]utils.UploadPayloadFile, len(bucket.Files))
				for i, file := range bucket.Files {
					data, err := os.ReadFile(file.Path)
					if err != nil {
						return err
					}
					encoded := base64.StdEncoding.EncodeToString(data)
					payload[i].Key = file.Hash
					payload[i].Value = encoded
					payload[i].Metadata.ContentType = file.ContentType
					payload[i].Base64 = true
				}
				payloadBytes, err := json.Marshal(payload)
				if err != nil {
					return err
				}
				headers := map[string]string{
					"Content-Type":  "application/json",
					"Authorization": "Bearer " + jwt,
				}

				_, err = utils.FetchResult[utils.UploadResponse]("/pages/assets/upload", "POST", headers, payloadBytes)
				if err != nil {
					if attempts < vars.MAX_UPLOAD_ATTEMPTS {
						time.Sleep(time.Second * time.Duration(1<<attempts))
						attempts++
						if apiErr, ok := err.(*utils.APIError); ok {
							// Check for gateway errors (e.g. 502, 503, 504)
							if apiErr.StatusCode == 502 || apiErr.StatusCode == 503 || apiErr.StatusCode == 504 {
								gatewayErrors++
								if gatewayErrors >= vars.MAX_UPLOAD_GATEWAY_ERRORS {
									attempts++
								}
								time.Sleep(time.Second * 5 * time.Duration(1<<gatewayErrors))
							} else if apiErr.StatusCode == 401 {
								if newJwt, err := fetchJwt(); err == nil {
									jwt = newJwt
								}
							}
						} else if expired, _ := utils.IsJwtExpired(jwt); expired {
							if newJwt, err := fetchJwt(); err == nil {
								jwt = newJwt
							}
						}
						return doUpload()
					}
					return err
				}
				return nil
			}

			// Limit concurrency per bucket.
			sem <- struct{}{}
			err := doUpload()
			<-sem
			if err != nil {
				mu.Lock()
				if uploadErr == nil {
					uploadErr = err
				}
				mu.Unlock()
			} else {
				mu.Lock()
				counter += len(bucket.Files)
				utils.IncUpTo(args.ProjectName, counter, len(args.FileMap))
				mu.Unlock()
			}
		}(bucket)
	}
	wg.Wait()

	if uploadErr != nil {
		return nil, uploadErr
	}

	uploadDuration := time.Since(start)
	skipped := len(args.FileMap) - len(missingHashes)
	skippedMessage := ""
	if skipped > 0 {
		skippedMessage = fmt.Sprintf("(%d already uploaded) ", skipped)
	}
	fmt.Printf("✨ Success! Uploaded %d files %s%s\n", len(sortedFiles), skippedMessage, utils.FormatTime(uploadDuration))

	// Upsert hashes.
	doUpsertHashes := func() error {
		payloadData := map[string][]string{
			"hashes": {},
		}
		for _, file := range files {
			payloadData["hashes"] = append(payloadData["hashes"], file.Hash)
		}
		payloadBytes, err := json.Marshal(payloadData)
		if err != nil {
			return err
		}
		headers := map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer " + jwt,
		}
		_, err = utils.FetchResult[any]("/pages/assets/upsert-hashes", "POST", headers, payloadBytes)
		if err != nil {
			time.Sleep(1 * time.Second)
			if apiErr, ok := err.(*utils.APIError); ok && apiErr.StatusCode == 401 {
				if newJwt, err := fetchJwt(); err == nil {
					jwt = newJwt
				}
			}

			_, upsertHashErr := utils.FetchResult[any]("/pages/assets/upsert-hashes", "POST", headers, payloadBytes)
			return upsertHashErr
		}
		return nil
	}

	if err := doUpsertHashes(); err != nil {
		fmt.Printf("Warning: Failed to update file hashes. Every upload appeared to succeed, but future deployments might re-upload files (this may slow subsequent deployments).")
	}

	// Build and return the manifest mapping file names (with a leading slash) to hashes.
	manifest := make(map[string]string)
	for fileName, file := range args.FileMap {
		manifest["/"+fileName] = file.Hash
	}
	return manifest, nil
}
