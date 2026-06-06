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

// csrfHeaderName is the request header 3x-ui expects for unsafe methods
// (mirrors web/session.CSRFHeaderName in the panel).
const csrfHeaderName = "X-CSRF-Token"

// Client communicates with the 3x-ui panel API.
//
// It supports two complementary auth mechanisms (3x-ui v3.2.8+):
//   - Session + CSRF: logs in with username/password, stores the session cookie
//     in a jar, and replays the session CSRF token on unsafe requests. Covers
//     every panel route.
//   - Bearer API token (optional): when set, sent on every request. The panel
//     accepts it for /panel/api/* routes and skips CSRF there. Routes under
//     /panel/* (settings, xray) are gated by session login, so they still need
//     credentials; the token alone does not unlock them.
type Client struct {
	baseURL  string
	username string
	password string
	apiToken string
	http     *http.Client
	logger   *slog.Logger

	mu        sync.Mutex
	loggedIn  bool
	csrfToken string
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
		apiToken: cfg.APIToken,
		http:     httpClient,
		logger:   logger,
	}
}

// fullURL builds a full URL for the given relative API path.
func (c *Client) fullURL(path string) string {
	return c.baseURL + strings.TrimLeft(path, "/")
}

func (c *Client) hasCredentials() bool {
	return c.username != "" && c.password != ""
}

func (c *Client) sessionActive() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.loggedIn
}

// fetchCSRFToken performs a GET on a csrf-token endpoint and returns the token.
// It does not mutate client state or take the lock, so it is safe to call from
// within login (which holds the lock) and from refreshCSRF (which does not).
func (c *Client) fetchCSRFToken(ctx context.Context, path string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.fullURL(path), nil)
	if err != nil {
		return "", fmt.Errorf("creating csrf request: %w", err)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return "", fmt.Errorf("csrf request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading csrf response: %w", err)
	}

	var r Response
	if err := json.Unmarshal(body, &r); err != nil {
		return "", fmt.Errorf("parsing csrf response: %w", err)
	}
	if !r.Success {
		return "", fmt.Errorf("csrf token request failed: %s", r.Msg)
	}

	var token string
	if err := json.Unmarshal(r.Obj, &token); err != nil {
		return "", fmt.Errorf("parsing csrf token value: %w", err)
	}
	if token == "" {
		return "", fmt.Errorf("empty csrf token returned")
	}
	return token, nil
}

// refreshCSRF re-fetches the CSRF token from the authenticated panel endpoint.
// Used to recover from a 403 caused by a stale token.
func (c *Client) refreshCSRF(ctx context.Context) error {
	token, err := c.fetchCSRFToken(ctx, "panel/csrf-token")
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.csrfToken = token
	c.mu.Unlock()
	return nil
}

// login authenticates with the panel, storing the session cookie in the jar
// and the session CSRF token on the client. Must be called with c.mu held.
func (c *Client) login(ctx context.Context) error {
	if !c.hasCredentials() {
		return fmt.Errorf("cannot establish a panel session: XUI_USERNAME/XUI_PASSWORD not set")
	}
	c.logger.Debug("logging in to 3x-ui panel")

	// /login is CSRF-protected, so obtain a public token first.
	token, err := c.fetchCSRFToken(ctx, "csrf-token")
	if err != nil {
		return fmt.Errorf("obtaining login csrf token: %w", err)
	}

	form := url.Values{
		"username": {c.username},
		"password": {c.password},
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.fullURL("login"), strings.NewReader(form.Encode()))
	if err != nil {
		return fmt.Errorf("creating login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set(csrfHeaderName, token)

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

	// The session id can rotate on login (SetLoginUser), which would invalidate
	// the pre-login CSRF token. Re-fetch it via the authenticated panel endpoint
	// so subsequent unsafe requests carry a token bound to the new session.
	if newToken, err := c.fetchCSRFToken(ctx, "panel/csrf-token"); err == nil {
		token = newToken
	} else {
		c.logger.Debug("post-login csrf refresh failed, using login token", "err", err)
	}

	c.csrfToken = token
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

// reAuth forces a new login (e.g., after session expiry detected via 403/404/3xx).
func (c *Client) reAuth(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.loggedIn = false
	return c.login(ctx)
}

// do performs an HTTP request with automatic auth and retry on session/CSRF expiry.
func (c *Client) do(ctx context.Context, method, path, contentType string, body []byte) (*Response, error) {
	// Session mode (no token): log in up front. In token mode we rely on the
	// Bearer header for /panel/api/* and only log in lazily if a route needs it.
	if c.apiToken == "" {
		if err := c.ensureAuth(ctx); err != nil {
			return nil, err
		}
	}

	resp, status, err := c.rawDo(ctx, method, path, contentType, body)
	if err != nil {
		return nil, err
	}
	if resp != nil {
		return resp, nil
	}

	// resp == nil → the panel signalled an auth problem:
	//   403       = CSRF token missing/stale (session-gated routes)
	//   404 / 3xx = no/expired session (or a /panel/* route reached in token mode)
	// Recovery requires panel credentials.
	if !c.hasCredentials() {
		return nil, c.authFailureError(path, status)
	}

	// A 403 on an active session usually just means the CSRF token went stale —
	// refreshing it is cheaper than a full re-login.
	if status == http.StatusForbidden && c.sessionActive() {
		if cerr := c.refreshCSRF(ctx); cerr == nil {
			resp, status, err = c.rawDo(ctx, method, path, contentType, body)
			if err != nil {
				return nil, err
			}
			if resp != nil {
				return resp, nil
			}
		}
	}

	// Fall back to a full re-login and retry once.
	c.logger.Debug("session expired, attempting re-auth", "path", path, "status", status)
	if err := c.reAuth(ctx); err != nil {
		return nil, fmt.Errorf("re-auth failed: %w", err)
	}
	resp, status, err = c.rawDo(ctx, method, path, contentType, body)
	if err != nil {
		return nil, err
	}
	if resp == nil {
		return nil, c.authFailureError(path, status)
	}
	return resp, nil
}

// authFailureError builds a descriptive error for an unrecoverable auth failure.
func (c *Client) authFailureError(path string, status int) error {
	if c.apiToken != "" && !c.hasCredentials() {
		return fmt.Errorf("request to %s failed (HTTP %d): this route needs a panel session, but only XUI_API_TOKEN is set — add XUI_USERNAME/XUI_PASSWORD for settings/xray tools, or verify the token", path, status)
	}
	if c.apiToken != "" {
		return fmt.Errorf("request to %s failed (HTTP %d) after re-auth — verify XUI_API_TOKEN and credentials", path, status)
	}
	return fmt.Errorf("request to %s failed (HTTP %d) after re-auth", path, status)
}

// rawDo performs a single HTTP request, attaching auth headers as configured.
// It returns (nil, status, nil) for auth-signalling statuses (403, 404, 3xx)
// so do() can decide how to recover.
func (c *Client) rawDo(ctx context.Context, method, path, contentType string, body []byte) (*Response, int, error) {
	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.fullURL(path), reqBody)
	if err != nil {
		return nil, 0, fmt.Errorf("creating request: %w", err)
	}
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if c.apiToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiToken)
	}
	if !isSafeMethod(method) {
		c.mu.Lock()
		token := c.csrfToken
		c.mu.Unlock()
		if token != "" {
			req.Header.Set(csrfHeaderName, token)
		}
	}

	c.logger.Debug("API request", "method", method, "path", path)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("HTTP %s %s: %w", method, path, err)
	}
	defer resp.Body.Close()

	status := resp.StatusCode

	// Auth signals — let do() handle recovery:
	//   404 (API routes) / 3xx redirect (panel routes) = expired/absent session
	//   403 = CSRF token rejected
	if status == http.StatusNotFound ||
		status == http.StatusForbidden ||
		(status >= 300 && status < 400) {
		return nil, status, nil
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, status, fmt.Errorf("reading response body: %w", err)
	}

	var apiResp Response
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		// Non-JSON response (e.g., file download)
		return &Response{
			Success: status == http.StatusOK,
			Msg:     resp.Status,
			Obj:     respBody,
		}, status, nil
	}

	c.logger.Debug("API response", "path", path, "success", apiResp.Success)
	return &apiResp, status, nil
}

// isSafeMethod reports whether the HTTP method is exempt from CSRF (read-only).
func isSafeMethod(method string) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions, http.MethodTrace:
		return true
	default:
		return false
	}
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
