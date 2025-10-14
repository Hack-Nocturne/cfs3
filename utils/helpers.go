package utils

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/Hack-Nocturne/cfs3/types"
	"github.com/Hack-Nocturne/cfs3/vars"
)

var mu sync.Mutex

// fetchResult makes an HTTP request to the given URL (appended to apiBaseURL)
// with the specified method, headers and body. It then decodes the JSON response into result.
func fetchResult[T any](url, method string, headers map[string]string, body []byte) (types.CFResponse[T], error) {
	client := &http.Client{}
	empty := types.CFResponse[T]{}
	req, err := http.NewRequest(method, vars.API_BASE_URL+url, bytes.NewReader(body))
	if err != nil {
		return empty, err
	}

	if headers == nil {
		headers = make(map[string]string)
	}

	headers["User-Agent"] = "wrangler/4.0.0"

	// Check if Authorization header exists, if not set default one
	if _, hasAuth := headers["Authorization"]; !hasAuth {
		headers["Authorization"] = "Bearer " + vars.CF_API_TOKEN
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := client.Do(req)
	if err != nil {
		return empty, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return empty, &types.APIError{StatusCode: resp.StatusCode, Message: string(respBody)}
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return empty, err
	}

	var response types.CFResponse[T]
	json.Unmarshal(respBody, &response)

	if !response.Success {
		defer logError(response.Errors)
	}

	return response, nil
}

func logError(errors []string) {
	mu.Lock()
	defer mu.Unlock()

	logEntry := struct {
		Timestamp time.Time `json:"timestamp"`
		Errors    any       `json:"errors"`
	}{
		Timestamp: time.Now(),
		Errors:    errors,
	}

	data, err := json.Marshal(logEntry)
	if err != nil {
		return
	}

	// Append to log file with file locking
	f, err := os.OpenFile("cf-errors.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	data = append(data, '\n')
	if _, err := f.Write(data); err != nil {
		return
	}
}

// isJwtExpired decodes the JWT (assumes a standard JWT with an "exp" claim)
// and returns whether it has expired.
func isJwtExpired(token string) (bool, error) {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return false, errors.New("invalid token format")
	}
	// Try decoding without padding.
	payload, err := base64.RawStdEncoding.DecodeString(parts[1])
	if err != nil {
		// Fallback to padded decoding.
		payload, err = base64.StdEncoding.DecodeString(parts[1])
		if err != nil {
			return false, err
		}
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		return false, err
	}
	expVal, ok := decoded["exp"]
	if !ok {
		return false, errors.New("token does not contain exp")
	}
	expFloat, ok := expVal.(float64)
	if !ok {
		return false, errors.New("invalid exp field in token")
	}
	now := float64(time.Now().Unix())
	return expFloat <= now, nil
}

// formatTime returns a formatted string for a duration.
func formatTime(duration time.Duration) string {
	return fmt.Sprintf("(%.2f sec)", duration.Seconds())
}

var (
	earthFrames = []string{"🌍", "🌎", "🌏"}
	frameIndex  int
	frameMu     sync.Mutex
)

// incUpTo prints a rotating-earth spinner plus “prefix: current/total” on one line.
func incUpTo(prefix string, current, total int) {
	frameMu.Lock()
	emoji := earthFrames[frameIndex]
	frameIndex = (frameIndex + 1) % len(earthFrames)
	frameMu.Unlock()

	// \r returns to the start of the line, \033[K clears to end of line
	fmt.Printf("\r\033[K%s %s: %d/%d", emoji, prefix, current, total)

	if current >= total {
		fmt.Println() // finish with newline
	}
}

func Clone[T any](src T) T {
	var dst T
	data, err := json.Marshal(src)
	if err != nil {
		panic(err) // Handle error appropriately in production code
	}
	err = json.Unmarshal(data, &dst)
	if err != nil {
		panic(err) // Handle error appropriately in production code
	}
	return dst
}
