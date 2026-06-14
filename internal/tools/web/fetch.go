package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ilmich/emmai/internal/client"
)

const (
	maxFetchBodyBytes = 512 * 1024 // 512KB
	fetchTimeoutSec   = 30
)

// FetchResponse is the response from the fetch_url tool
type FetchResponse struct {
	Success     bool   `json:"success"`
	URL         string `json:"url"`
	StatusCode  int    `json:"status_code,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	Body        string `json:"body,omitempty"`
	Truncated   bool   `json:"truncated,omitempty"`
	Error       string `json:"error,omitempty"`
	Hint        string `json:"hint,omitempty"`
}

// FetchExecutor handles URL fetch operations
type FetchExecutor struct {
	httpClient *http.Client
}

// NewFetchExecutor creates a new fetch executor
func NewFetchExecutor() *FetchExecutor {
	return &FetchExecutor{
		httpClient: &http.Client{
			Timeout: fetchTimeoutSec * time.Second,
		},
	}
}

// NewFetchURLTool returns the fetch_url tool definition
func NewFetchURLTool() client.Tool {
	return client.NewFunctionTool(
		"fetch_url",
		"Fetch content from a URL (web page, API, documentation). Returns the response body as plain text. HTML is stripped to readable text. Use to look up documentation, check API references, read GitHub issues, or access any public URL.",
		map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"url": map[string]interface{}{
					"type":        "string",
					"description": "The URL to fetch (must be http:// or https://)",
				},
				"max_length": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of characters to return (default: 20000, max: 50000)",
				},
			},
			"required": []string{"url"},
		},
	)
}

// HandleFetchURL executes the fetch_url tool
func (e *FetchExecutor) HandleFetchURL(args map[string]interface{}) (string, error) {
	rawURL, _ := args["url"].(string)
	if rawURL == "" {
		return e.errorResponse("", "url is required", "Provide a valid http:// or https:// URL")
	}

	maxLen := 20000
	if v, ok := args["max_length"].(float64); ok && v > 0 {
		maxLen = int(v)
		if maxLen > 50000 {
			maxLen = 50000
		}
	}

	parsed, err := url.Parse(rawURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return e.errorResponse(rawURL, "invalid URL: must start with http:// or https://", "Check the URL format")
	}

	ctx, cancel := context.WithTimeout(context.Background(), fetchTimeoutSec*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return e.errorResponse(rawURL, fmt.Sprintf("failed to create request: %v", err), "")
	}
	req.Header.Set("User-Agent", "emmai-agent/1.0")
	req.Header.Set("Accept", "text/html,text/plain,application/json,*/*")

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return e.errorResponse(rawURL, fmt.Sprintf("request failed: %v", err), "Check network connectivity")
	}
	defer resp.Body.Close()

	limited := io.LimitReader(resp.Body, maxFetchBodyBytes)
	bodyBytes, err := io.ReadAll(limited)
	if err != nil {
		return e.errorResponse(rawURL, fmt.Sprintf("failed to read response: %v", err), "")
	}

	contentType := resp.Header.Get("Content-Type")
	body := string(bodyBytes)
	truncated := len(bodyBytes) >= maxFetchBodyBytes

	// Strip HTML tags if content is HTML
	if strings.Contains(contentType, "text/html") {
		body = stripHTML(body)
	}

	// Ensure valid UTF-8
	if !utf8.ValidString(body) {
		body = strings.ToValidUTF8(body, "")
	}

	if len(body) > maxLen {
		body = body[:maxLen]
		truncated = true
	}

	result := FetchResponse{
		Success:     resp.StatusCode >= 200 && resp.StatusCode < 300,
		URL:         rawURL,
		StatusCode:  resp.StatusCode,
		ContentType: contentType,
		Body:        body,
		Truncated:   truncated,
	}
	if !result.Success {
		result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		result.Hint = "The server returned an error status"
	}

	out, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("failed to encode response: %w", err)
	}
	return string(out), nil
}

func (e *FetchExecutor) errorResponse(rawURL, msg, hint string) (string, error) {
	r := FetchResponse{
		Success: false,
		URL:     rawURL,
		Error:   msg,
		Hint:    hint,
	}
	out, err := json.Marshal(r)
	if err != nil {
		return "", fmt.Errorf("failed to encode error response: %w", err)
	}
	return string(out), nil
}

var (
	reScript  = regexp.MustCompile(`(?is)<(script|style)[^>]*>.*?</(script|style)>`)
	reTag     = regexp.MustCompile(`<[^>]+>`)
	reSpaces  = regexp.MustCompile(`[ \t]{2,}`)
	reNewline = regexp.MustCompile(`\n{3,}`)
)

// stripHTML removes HTML tags and collapses whitespace into readable plain text.
func stripHTML(html string) string {
	s := reScript.ReplaceAllString(html, "")
	s = reTag.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, "&amp;", "&")
	s = strings.ReplaceAll(s, "&lt;", "<")
	s = strings.ReplaceAll(s, "&gt;", ">")
	s = strings.ReplaceAll(s, "&quot;", `"`)
	s = strings.ReplaceAll(s, "&#39;", "'")
	s = strings.ReplaceAll(s, "&nbsp;", " ")
	s = reSpaces.ReplaceAllString(s, " ")
	lines := strings.Split(s, "\n")
	var out []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			out = append(out, l)
		}
	}
	s = strings.Join(out, "\n")
	s = reNewline.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}
