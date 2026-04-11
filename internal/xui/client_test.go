package xui

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync/atomic"
	"testing"

	"github.com/pyworkload/3x-ui-mcp/internal/config"
)

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

func TestNewClient_CorrectConfig(t *testing.T) {
	cfg := &config.Config{
		Host:     "http://example.com",
		BasePath: "/panel/",
		Username: "user1",
		Password: "pass1",
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
}

func TestLogin_Failure(t *testing.T) {
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
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

func TestAutoAuth_OnFirstRequest(t *testing.T) {
	var loginCalled atomic.Int32

	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
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

func TestGet_Method(t *testing.T) {
	var capturedMethod string

	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
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

func TestPostJSON_Method(t *testing.T) {
	var capturedMethod string
	var capturedContentType string
	var capturedBody map[string]any

	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
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
