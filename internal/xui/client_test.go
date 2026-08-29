package xui

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/pyworkload/3x-ui-mcp/internal/config"
)

const testCSRFToken = "test-csrf-token"

// writeCSRFIfRequested answers the public (/csrf-token) and authenticated
// (/panel/csrf-token) CSRF endpoints. Returns true if it handled the request.
func writeCSRFIfRequested(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path == "/csrf-token" || r.URL.Path == "/panel/csrf-token" {
		_ = json.NewEncoder(w).Encode(Response{Success: true, Obj: json.RawMessage(`"` + testCSRFToken + `"`)})
		return true
	}
	return false
}

// newTestServer creates an httptest.Server and a Client wired to it.
// The handler is called for every request.
func newTestServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *Client) {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	cfg := &config.Config{
		Host:     ts.URL,
		BasePath: "/",
		Username: "admin",
		Password: "admin",
	}

	logger := slog.Default()
	client := NewClient(cfg, logger)
	return ts, client
}

// newTestServerWithToken wires a Client configured for Bearer-token auth.
func newTestServerWithToken(t *testing.T, token string, handler http.HandlerFunc) (*httptest.Server, *Client) {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)

	cfg := &config.Config{
		Host:     ts.URL,
		BasePath: "/",
		Username: "admin",
		Password: "admin",
		APIToken: token,
	}
	client := NewClient(cfg, slog.Default())
	return ts, client
}

func TestNewClient_CorrectConfig(t *testing.T) {
	cfg := &config.Config{
		Host:     "http://example.com",
		BasePath: "/panel/",
		Username: "user1",
		Password: "pass1",
		APIToken: "tok",
	}
	logger := slog.Default()
	c := NewClient(cfg, logger)

	if c.baseURL != "http://example.com/panel/" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "http://example.com/panel/")
	}
	if c.username != "user1" {
		t.Errorf("username = %q, want %q", c.username, "user1")
	}
	if c.password != "pass1" {
		t.Errorf("password = %q, want %q", c.password, "pass1")
	}
	if c.apiToken != "tok" {
		t.Errorf("apiToken = %q, want %q", c.apiToken, "tok")
	}
	if c.loggedIn {
		t.Error("expected loggedIn to be false initially")
	}
	if c.http == nil {
		t.Error("expected http client to be non-nil")
	}
	if c.http.Jar == nil {
		t.Error("expected cookie jar to be non-nil")
	}
}

func TestLogin_Success(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if writeCSRFIfRequested(w, r) {
			return
		}
		if r.URL.Path == "/login" {
			resp := Response{Success: true, Msg: "ok"}
			json.NewEncoder(w).Encode(resp)
			return
		}
		t.Errorf("unexpected request path: %s", r.URL.Path)
	})

	err := client.login(context.Background())
	if err != nil {
		t.Fatalf("login returned error: %v", err)
	}
	if !client.loggedIn {
		t.Error("expected loggedIn to be true after successful login")
	}
	if client.csrfToken != testCSRFToken {
		t.Errorf("csrfToken = %q, want %q", client.csrfToken, testCSRFToken)
	}
}

func TestLogin_Failure(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if writeCSRFIfRequested(w, r) {
			return
		}
		if r.URL.Path == "/login" {
			resp := Response{Success: false, Msg: "invalid credentials"}
			json.NewEncoder(w).Encode(resp)
			return
		}
		t.Errorf("unexpected request path: %s", r.URL.Path)
	})

	err := client.login(context.Background())
	if err == nil {
		t.Fatal("expected error for failed login, got nil")
	}
	if client.loggedIn {
		t.Error("expected loggedIn to be false after failed login")
	}
}

func TestLogin_SendsCSRFHeader(t *testing.T) {
	var loginCSRF string
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if writeCSRFIfRequested(w, r) {
			return
		}
		if r.URL.Path == "/login" {
			loginCSRF = r.Header.Get(csrfHeaderName)
			json.NewEncoder(w).Encode(Response{Success: true})
			return
		}
	})

	if err := client.login(context.Background()); err != nil {
		t.Fatalf("login error: %v", err)
	}
	if loginCSRF != testCSRFToken {
		t.Errorf("login X-CSRF-Token = %q, want %q", loginCSRF, testCSRFToken)
	}
}

func TestAutoAuth_OnFirstRequest(t *testing.T) {
	var loginCalled atomic.Int32

	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if writeCSRFIfRequested(w, r) {
			return
		}
		switch r.URL.Path {
		case "/login":
			loginCalled.Add(1)
			resp := Response{Success: true, Msg: "ok"}
			json.NewEncoder(w).Encode(resp)
		case "/panel/api/inbounds/list":
			resp := Response{Success: true, Msg: "list ok", Obj: json.RawMessage(`[]`)}
			json.NewEncoder(w).Encode(resp)
		default:
			t.Errorf("unexpected request path: %s", r.URL.Path)
		}
	})

	resp, err := client.Get(context.Background(), "panel/api/inbounds/list")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if !resp.Success {
		t.Error("expected success response")
	}
	if loginCalled.Load() != 1 {
		t.Errorf("login called %d times, want 1", loginCalled.Load())
	}
}

func TestSessionExpiry_RetriesAfter404(t *testing.T) {
	var requestCount atomic.Int32
	var loginCount atomic.Int32

	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if writeCSRFIfRequested(w, r) {
			return
		}
		switch r.URL.Path {
		case "/login":
			loginCount.Add(1)
			resp := Response{Success: true, Msg: "ok"}
			json.NewEncoder(w).Encode(resp)
		case "/panel/api/data":
			count := requestCount.Add(1)
			if count == 1 {
				// First attempt: simulate session expiry with 404
				w.WriteHeader(http.StatusNotFound)
				return
			}
			// Second attempt after re-auth: success
			resp := Response{Success: true, Msg: "data ok", Obj: json.RawMessage(`{"key":"value"}`)}
			json.NewEncoder(w).Encode(resp)
		default:
			t.Errorf("unexpected request path: %s", r.URL.Path)
		}
	})

	resp, err := client.Get(context.Background(), "panel/api/data")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if !resp.Success {
		t.Error("expected success response after retry")
	}
	// Initial login + re-auth = 2 logins
	if loginCount.Load() != 2 {
		t.Errorf("login called %d times, want 2", loginCount.Load())
	}
	// 2 requests to the data endpoint
	if requestCount.Load() != 2 {
		t.Errorf("data endpoint called %d times, want 2", requestCount.Load())
	}
}

func TestRedirectDetection_SessionExpired(t *testing.T) {
	var requestCount atomic.Int32
	var loginCount atomic.Int32

	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if writeCSRFIfRequested(w, r) {
			return
		}
		switch r.URL.Path {
		case "/login":
			loginCount.Add(1)
			resp := Response{Success: true, Msg: "ok"}
			json.NewEncoder(w).Encode(resp)
		case "/panel/api/resource":
			count := requestCount.Add(1)
			if count == 1 {
				// Simulate session expiry via 307 redirect
				w.Header().Set("Location", "/login")
				w.WriteHeader(http.StatusTemporaryRedirect)
				return
			}
			resp := Response{Success: true, Msg: "resource ok"}
			json.NewEncoder(w).Encode(resp)
		default:
			t.Errorf("unexpected request path: %s", r.URL.Path)
		}
	})

	resp, err := client.Get(context.Background(), "panel/api/resource")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if !resp.Success {
		t.Error("expected success response after redirect-triggered re-auth")
	}
	if loginCount.Load() != 2 {
		t.Errorf("login called %d times, want 2 (initial + re-auth)", loginCount.Load())
	}
}

func TestCSRF_403RefreshesTokenAndRetries(t *testing.T) {
	var requestCount atomic.Int32
	var loginCount atomic.Int32

	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if writeCSRFIfRequested(w, r) {
			return
		}
		switch r.URL.Path {
		case "/login":
			loginCount.Add(1)
			json.NewEncoder(w).Encode(Response{Success: true})
		case "/panel/api/clients/add":
			count := requestCount.Add(1)
			if count == 1 {
				// Stale CSRF token → 403 (no body, like gin's AbortWithStatus)
				w.WriteHeader(http.StatusForbidden)
				return
			}
			json.NewEncoder(w).Encode(Response{Success: true, Msg: "added"})
		default:
			t.Errorf("unexpected request path: %s", r.URL.Path)
		}
	})

	resp, err := client.PostJSON(context.Background(), "panel/api/clients/add", map[string]string{"x": "y"})
	if err != nil {
		t.Fatalf("PostJSON returned error: %v", err)
	}
	if !resp.Success {
		t.Error("expected success after CSRF refresh + retry")
	}
	// A 403 should be recovered by refreshing CSRF, NOT by a full re-login.
	if loginCount.Load() != 1 {
		t.Errorf("login called %d times, want 1 (no re-login on 403)", loginCount.Load())
	}
	if requestCount.Load() != 2 {
		t.Errorf("endpoint called %d times, want 2", requestCount.Load())
	}
}

func TestGet_Method(t *testing.T) {
	var capturedMethod string

	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if writeCSRFIfRequested(w, r) {
			return
		}
		if r.URL.Path == "/login" {
			json.NewEncoder(w).Encode(Response{Success: true})
			return
		}
		capturedMethod = r.Method
		json.NewEncoder(w).Encode(Response{Success: true, Msg: "ok"})
	})

	_, err := client.Get(context.Background(), "api/test")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if capturedMethod != http.MethodGet {
		t.Errorf("method = %q, want %q", capturedMethod, http.MethodGet)
	}
}

func TestCSRFToken_OnUnsafeMethodsOnly(t *testing.T) {
	var getCSRF, postCSRF string
	var sawGet, sawPost atomic.Bool

	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if writeCSRFIfRequested(w, r) {
			return
		}
		if r.URL.Path == "/login" {
			json.NewEncoder(w).Encode(Response{Success: true})
			return
		}
		switch r.Method {
		case http.MethodGet:
			sawGet.Store(true)
			getCSRF = r.Header.Get(csrfHeaderName)
		case http.MethodPost:
			sawPost.Store(true)
			postCSRF = r.Header.Get(csrfHeaderName)
		}
		json.NewEncoder(w).Encode(Response{Success: true, Msg: "ok"})
	})

	if _, err := client.Get(context.Background(), "panel/api/x"); err != nil {
		t.Fatalf("Get error: %v", err)
	}
	if _, err := client.Post(context.Background(), "panel/api/y"); err != nil {
		t.Fatalf("Post error: %v", err)
	}

	if !sawGet.Load() || !sawPost.Load() {
		t.Fatal("expected both GET and POST to reach the server")
	}
	if getCSRF != "" {
		t.Errorf("GET should not carry a CSRF token, got %q", getCSRF)
	}
	if postCSRF != testCSRFToken {
		t.Errorf("POST X-CSRF-Token = %q, want %q", postCSRF, testCSRFToken)
	}
}

func TestPostJSON_Method(t *testing.T) {
	var capturedMethod string
	var capturedContentType string
	var capturedBody map[string]any

	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if writeCSRFIfRequested(w, r) {
			return
		}
		if r.URL.Path == "/login" {
			json.NewEncoder(w).Encode(Response{Success: true})
			return
		}
		capturedMethod = r.Method
		capturedContentType = r.Header.Get("Content-Type")
		json.NewDecoder(r.Body).Decode(&capturedBody)
		json.NewEncoder(w).Encode(Response{Success: true, Msg: "created"})
	})

	payload := map[string]string{"name": "test"}
	_, err := client.PostJSON(context.Background(), "api/create", payload)
	if err != nil {
		t.Fatalf("PostJSON returned error: %v", err)
	}
	if capturedMethod != http.MethodPost {
		t.Errorf("method = %q, want %q", capturedMethod, http.MethodPost)
	}
	if capturedContentType != "application/json" {
		t.Errorf("content-type = %q, want %q", capturedContentType, "application/json")
	}
	if capturedBody["name"] != "test" {
		t.Errorf("body[name] = %v, want %q", capturedBody["name"], "test")
	}
}

func TestPostForm_Method(t *testing.T) {
	var capturedMethod string
	var capturedContentType string
	var capturedFormValue string

	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if writeCSRFIfRequested(w, r) {
			return
		}
		if r.URL.Path == "/login" {
			json.NewEncoder(w).Encode(Response{Success: true})
			return
		}
		capturedMethod = r.Method
		capturedContentType = r.Header.Get("Content-Type")
		r.ParseForm()
		capturedFormValue = r.FormValue("key")
		json.NewEncoder(w).Encode(Response{Success: true, Msg: "submitted"})
	})

	data := url.Values{"key": {"value123"}}
	_, err := client.PostForm(context.Background(), "api/submit", data)
	if err != nil {
		t.Fatalf("PostForm returned error: %v", err)
	}
	if capturedMethod != http.MethodPost {
		t.Errorf("method = %q, want %q", capturedMethod, http.MethodPost)
	}
	if capturedContentType != "application/x-www-form-urlencoded" {
		t.Errorf("content-type = %q, want %q", capturedContentType, "application/x-www-form-urlencoded")
	}
	if capturedFormValue != "value123" {
		t.Errorf("form value key = %q, want %q", capturedFormValue, "value123")
	}
}

func TestPost_NoBody(t *testing.T) {
	var capturedMethod string
	var capturedContentLength int64

	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if writeCSRFIfRequested(w, r) {
			return
		}
		if r.URL.Path == "/login" {
			json.NewEncoder(w).Encode(Response{Success: true})
			return
		}
		capturedMethod = r.Method
		capturedContentLength = r.ContentLength
		json.NewEncoder(w).Encode(Response{Success: true, Msg: "ok"})
	})

	_, err := client.Post(context.Background(), "api/action")
	if err != nil {
		t.Fatalf("Post returned error: %v", err)
	}
	if capturedMethod != http.MethodPost {
		t.Errorf("method = %q, want %q", capturedMethod, http.MethodPost)
	}
	if capturedContentLength > 0 {
		t.Errorf("content-length = %d, want 0 or -1 (no body)", capturedContentLength)
	}
}

func TestFullURL_Construction(t *testing.T) {
	cfg := &config.Config{
		Host:     "http://example.com",
		BasePath: "/panel/",
		Username: "u",
		Password: "p",
	}
	c := NewClient(cfg, slog.Default())

	tests := []struct {
		path     string
		expected string
	}{
		{"api/inbounds", "http://example.com/panel/api/inbounds"},
		{"/api/inbounds", "http://example.com/panel/api/inbounds"},
		{"login", "http://example.com/panel/login"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := c.fullURL(tt.path)
			if got != tt.expected {
				t.Errorf("fullURL(%q) = %q, want %q", tt.path, got, tt.expected)
			}
		})
	}
}

func TestNonJSON_Response(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if writeCSRFIfRequested(w, r) {
			return
		}
		if r.URL.Path == "/login" {
			json.NewEncoder(w).Encode(Response{Success: true})
			return
		}
		// Return plain text (non-JSON)
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "some plain text data")
	})

	resp, err := client.Get(context.Background(), "api/download")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if !resp.Success {
		t.Error("expected success=true for 200 non-JSON response")
	}
}

func TestLogin_SendsCredentials(t *testing.T) {
	var capturedUsername string
	var capturedPassword string

	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if writeCSRFIfRequested(w, r) {
			return
		}
		if r.URL.Path == "/login" {
			r.ParseForm()
			capturedUsername = r.FormValue("username")
			capturedPassword = r.FormValue("password")
			json.NewEncoder(w).Encode(Response{Success: true})
			return
		}
	})

	err := client.login(context.Background())
	if err != nil {
		t.Fatalf("login error: %v", err)
	}
	if capturedUsername != "admin" {
		t.Errorf("username = %q, want %q", capturedUsername, "admin")
	}
	if capturedPassword != "admin" {
		t.Errorf("password = %q, want %q", capturedPassword, "admin")
	}
}

func TestDo_FailsAfterReauthStill404(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if writeCSRFIfRequested(w, r) {
			return
		}
		if r.URL.Path == "/login" {
			json.NewEncoder(w).Encode(Response{Success: true})
			return
		}
		// Always return 404 even after re-auth
		w.WriteHeader(http.StatusNotFound)
	})

	_, err := client.Get(context.Background(), "api/missing")
	if err == nil {
		t.Fatal("expected error when endpoint returns 404 even after re-auth, got nil")
	}
}

func TestBearerMode_SetsAuthHeaderNoLogin(t *testing.T) {
	var authHeader string
	var loginCalled atomic.Int32

	_, client := newTestServerWithToken(t, "secret-token", func(w http.ResponseWriter, r *http.Request) {
		if writeCSRFIfRequested(w, r) {
			return
		}
		if r.URL.Path == "/login" {
			loginCalled.Add(1)
			json.NewEncoder(w).Encode(Response{Success: true})
			return
		}
		authHeader = r.Header.Get("Authorization")
		json.NewEncoder(w).Encode(Response{Success: true, Msg: "ok", Obj: json.RawMessage(`[]`)})
	})

	resp, err := client.Get(context.Background(), "panel/api/inbounds/list")
	if err != nil {
		t.Fatalf("Get returned error: %v", err)
	}
	if !resp.Success {
		t.Error("expected success in bearer mode")
	}
	if authHeader != "Bearer secret-token" {
		t.Errorf("Authorization = %q, want %q", authHeader, "Bearer secret-token")
	}
	// A valid token for a /panel/api/* route must not trigger a session login.
	if loginCalled.Load() != 0 {
		t.Errorf("login called %d times, want 0 in bearer mode", loginCalled.Load())
	}
}

func TestBearerMode_FallsBackToSessionForPanelRoutes(t *testing.T) {
	var requestCount atomic.Int32
	var loginCount atomic.Int32

	_, client := newTestServerWithToken(t, "secret-token", func(w http.ResponseWriter, r *http.Request) {
		if writeCSRFIfRequested(w, r) {
			return
		}
		switch r.URL.Path {
		case "/login":
			loginCount.Add(1)
			json.NewEncoder(w).Encode(Response{Success: true})
		case "/panel/setting/all":
			count := requestCount.Add(1)
			if count == 1 {
				// No session yet → panel redirects (Bearer doesn't unlock /panel/*).
				w.Header().Set("Location", "/")
				w.WriteHeader(http.StatusTemporaryRedirect)
				return
			}
			json.NewEncoder(w).Encode(Response{Success: true, Msg: "settings"})
		default:
			t.Errorf("unexpected request path: %s", r.URL.Path)
		}
	})

	resp, err := client.Post(context.Background(), "panel/setting/all")
	if err != nil {
		t.Fatalf("Post returned error: %v", err)
	}
	if !resp.Success {
		t.Error("expected success after lazy session login")
	}
	if loginCount.Load() != 1 {
		t.Errorf("login called %d times, want 1 (lazy session for /panel/* route)", loginCount.Load())
	}
	if requestCount.Load() != 2 {
		t.Errorf("endpoint called %d times, want 2", requestCount.Load())
	}
}

func TestTokenOnly_NoCredentialsPanelRouteErrors(t *testing.T) {
	cfg := &config.Config{
		Host:     "", // set below
		BasePath: "/",
		APIToken: "secret-token",
		// No username/password: token-only deployment.
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if writeCSRFIfRequested(w, r) {
			return
		}
		// /panel/* route with no session and a token that the panel ignores here.
		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusTemporaryRedirect)
	}))
	t.Cleanup(ts.Close)
	cfg.Host = ts.URL
	client := NewClient(cfg, slog.Default())

	_, err := client.Post(context.Background(), "panel/setting/all")
	if err == nil {
		t.Fatal("expected an error for a session-gated route without credentials")
	}
}

// GET on a route the panel doesn't have lands on the SPA shell: 200 with
// index.html. That must not reach the caller as a successful API response.
func TestSPAShell_NotReportedAsSuccess(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if writeCSRFIfRequested(w, r) {
			return
		}
		if r.URL.Path == "/login" {
			json.NewEncoder(w).Encode(Response{Success: true})
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "<!doctype html>\n<html lang=\"en\"><head><title>3x-ui</title></head></html>")
	})

	_, err := client.Get(context.Background(), "panel/api/inbounds/list")
	if err == nil {
		t.Fatal("expected an error when the panel returns the SPA shell, got nil")
	}
}

// A 404 on a route that moved in v3.3.0 should say so, not just report an auth
// failure — the usual cause is a panel older than v3.3.0.
func TestRelocatedRoute_404MentionsPanelVersion(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if writeCSRFIfRequested(w, r) {
			return
		}
		if r.URL.Path == "/login" {
			json.NewEncoder(w).Encode(Response{Success: true})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	_, err := client.Post(context.Background(), "panel/api/xray/")
	if err == nil {
		t.Fatal("expected an error for a route the panel does not serve, got nil")
	}
	if !strings.Contains(err.Error(), "v3.3.0") {
		t.Errorf("error %q should mention the v3.3.0 route move", err)
	}
}

func TestIsRelocatedRoute(t *testing.T) {
	cases := []struct {
		path string
		want bool
	}{
		{"panel/api/xray/", true},
		{"panel/api/xray/getOutboundsTraffic", true},
		{"panel/api/setting/all", true},
		{"/panel/api/setting/update", true},
		{"panel/api/inbounds/list", false},
		{"panel/csrf-token", false},
	}
	for _, tc := range cases {
		if got := isRelocatedRoute(tc.path); got != tc.want {
			t.Errorf("isRelocatedRoute(%q) = %v, want %v", tc.path, got, tc.want)
		}
	}
}

// An unconfigured client is what a registry crawler or a first-run misconfigured
// desktop client gets: the server started, the tools are listed, and the failure
// has to arrive on the call itself — without touching the network.
func TestUnconfigured_CallFailsWithConfigError(t *testing.T) {
	t.Setenv("XUI_HOST", "")
	t.Setenv("XUI_USERNAME", "")
	t.Setenv("XUI_PASSWORD", "")
	t.Setenv("XUI_API_TOKEN", "")

	cfg, err := config.Load()
	if err == nil {
		t.Fatal("expected the empty environment to fail validation")
	}

	client := NewClient(cfg, slog.Default())
	_, err = client.Get(context.Background(), "panel/api/inbounds/list")
	if err == nil {
		t.Fatal("expected an error from an unconfigured client, got nil")
	}
	if !strings.Contains(err.Error(), "XUI_HOST") {
		t.Errorf("error should name the missing setting, got: %v", err)
	}
}
