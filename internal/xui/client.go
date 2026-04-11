package xui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/pyworkload/3x-ui-mcp/internal/config"
)

// Client communicates with the 3x-ui panel API.
// It manages session-based authentication transparently.
type Client struct {
	baseURL  string
	username string
	password string
	http     *http.Client
	logger   *slog.Logger

	mu       sync.Mutex
	loggedIn bool
}

// NewClient creates a new XUI API client.
func NewClient(cfg *config.Config, logger *slog.Logger) *Client {
	jar, _ := cookiejar.New(nil)
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Jar:     jar,
		// Don't follow redirects — panel routes return 307 on auth failure.
		// We detect this and trigger re-auth instead of following to the login page.
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	return &Client{
		baseURL:  cfg.BaseURL(),
		username: cfg.Username,
		password: cfg.Password,
		http:     httpClient,
		logger:   logger,
	}
}

// fullURL builds a full URL for the given relative API path.
func (c *Client) fullURL(path string) string {
	return c.baseURL + strings.TrimLeft(path, "/")
}

// login authenticates with the panel, storing the session cookie in the jar.
func (c *Client) login(ctx context.Context) error {
	c.logger.Debug("logging in to 3x-ui panel")

	form := url.Values{
		"username": {c.username},
		"password": {c.password},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.fullURL("login"), strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("creating login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("login request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading login response: %w", err)
	}

	var apiResp Response
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return fmt.Errorf("parsing login response: %w", err)
	}
	if !apiResp.Success {
		return fmt.Errorf("login failed: %s", apiResp.Msg)
	}

	c.loggedIn = true
	c.logger.Debug("login successful")
	return nil
}

// ensureAuth guarantees an active session before making API calls.
func (c *Client) ensureAuth(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !c.loggedIn {
		return c.login(ctx)
	}
	return nil
}

// reAuth forces a new login (e.g., after session expiry detected via 404).
func (c *Client) reAuth(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.loggedIn = false
	return c.login(ctx)
}

// do performs an HTTP request with automatic auth and retry on session expiry.
func (c *Client) do(ctx context.Context, method, path, contentType string, body []byte) (*Response, error) {
	if err := c.ensureAuth(ctx); err != nil {
		return nil, err
	}

	resp, err := c.rawDo(ctx, method, path, contentType, body)
	if err != nil {
		return nil, err
	}

	// nil = session expired (404 from API routes, or 3xx redirect from panel routes)
	if resp == nil {
		c.logger.Debug("session expired, attempting re-auth", "path", path)
		if err := c.reAuth(ctx); err != nil {
			return nil, fmt.Errorf("re-auth failed: %w", err)
		}
		resp, err = c.rawDo(ctx, method, path, contentType, body)
		if err != nil {
			return nil, err
		}
		if resp == nil {
			return nil, fmt.Errorf("request to %s failed after re-auth (404)", path)
		}
	}

	return resp, nil
}

// rawDo performs a single HTTP request. Returns nil Response on 404.
func (c *Client) rawDo(ctx context.Context, method, path, contentType string, body []byte) (*Response, error) {
	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.fullURL(path), reqBody)
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	c.logger.Debug("API request", "method", method, "path", path)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	// 404 (API routes) or 3xx redirect (panel routes) = session expired
	if resp.StatusCode == http.StatusNotFound ||
		(resp.StatusCode >= 300 && resp.StatusCode < 400) {
		return nil, nil
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %w", err)
	}

	var apiResp Response
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		// Non-JSON response (e.g., file download)
		return &Response{
			Success: resp.StatusCode == http.StatusOK,
			Msg:     resp.Status,
			Obj:     respBody,
		}, nil
	}

	c.logger.Debug("API response", "path", path, "success", apiResp.Success)
	return &apiResp, nil
}

// Get performs an authenticated GET request.
func (c *Client) Get(ctx context.Context, path string) (*Response, error) {
	return c.do(ctx, http.MethodGet, path, "", nil)
}

// PostJSON performs an authenticated POST with a JSON body.
func (c *Client) PostJSON(ctx context.Context, path string, data any) (*Response, error) {
	body, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshaling JSON body: %w", err)
	}
	return c.do(ctx, http.MethodPost, path, "application/json", body)
}

// PostForm performs an authenticated POST with form-encoded body.
func (c *Client) PostForm(ctx context.Context, path string, data url.Values) (*Response, error) {
	return c.do(ctx, http.MethodPost, path, "application/x-www-form-urlencoded", []byte(data.Encode()))
}

// Post performs an authenticated POST with no body.
func (c *Client) Post(ctx context.Context, path string) (*Response, error) {
	return c.do(ctx, http.MethodPost, path, "", nil)
}
