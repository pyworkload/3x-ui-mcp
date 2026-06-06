package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pyworkload/3x-ui-mcp/internal/config"
	"github.com/pyworkload/3x-ui-mcp/internal/xui"

	"github.com/mark3labs/mcp-go/mcp"
)

func writeCSRF(w http.ResponseWriter, r *http.Request) bool {
	if r.URL.Path == "/csrf-token" || r.URL.Path == "/panel/csrf-token" {
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "obj": "csrf"})
		return true
	}
	return false
}

func newClientHandler(t *testing.T, handler http.HandlerFunc) (*clientHandler, *httptest.Server) {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	cfg := &config.Config{Host: ts.URL, BasePath: "/", Username: "admin", Password: "admin"}
	return &clientHandler{client: xui.NewClient(cfg, slog.Default())}, ts
}

func req(args map[string]any) mcp.CallToolRequest {
	return mcp.CallToolRequest{Params: mcp.CallToolParams{Arguments: args}}
}

// TestAddClient_BuildsPayload exercises the inbound_ids array parsing and the
// ClientCreatePayload shape end-to-end against a mock panel.
func TestAddClient_BuildsPayload(t *testing.T) {
	var gotPayload xui.ClientCreatePayload

	h, _ := newClientHandler(t, func(w http.ResponseWriter, r *http.Request) {
		if writeCSRF(w, r) {
			return
		}
		switch r.URL.Path {
		case "/login":
			json.NewEncoder(w).Encode(map[string]any{"success": true})
		case "/panel/api/clients/add":
			if err := json.NewDecoder(r.Body).Decode(&gotPayload); err != nil {
				t.Errorf("decoding payload: %v", err)
			}
			json.NewEncoder(w).Encode(map[string]any{"success": true, "msg": "added"})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	})

	result, err := h.add(context.Background(), req(map[string]any{
		"inbound_ids": []any{float64(1), float64(3)},
		"email":       "user@example.com",
		"total_gb":    float64(5),
	}))
	if err != nil {
		t.Fatalf("add returned go error: %v", err)
	}
	if result.IsError {
		t.Fatalf("add returned tool error: %+v", result.Content)
	}

	if len(gotPayload.InboundIds) != 2 || gotPayload.InboundIds[0] != 1 || gotPayload.InboundIds[1] != 3 {
		t.Errorf("inboundIds = %v, want [1 3]", gotPayload.InboundIds)
	}
	if gotPayload.Client.Email != "user@example.com" {
		t.Errorf("email = %q, want %q", gotPayload.Client.Email, "user@example.com")
	}
	if gotPayload.Client.ID == "" {
		t.Error("expected an auto-generated UUID in the payload")
	}
	const wantBytes = 5 * 1073741824
	if gotPayload.Client.TotalGB != wantBytes {
		t.Errorf("totalGB = %d, want %d (5 GB in bytes)", gotPayload.Client.TotalGB, wantBytes)
	}
}

func TestAddClient_RequiresInboundIDs(t *testing.T) {
	h, _ := newClientHandler(t, func(w http.ResponseWriter, r *http.Request) {
		if writeCSRF(w, r) {
			return
		}
		t.Errorf("no request should be sent when inbound_ids is missing, got %s", r.URL.Path)
	})

	result, err := h.add(context.Background(), req(map[string]any{"email": "user@example.com"}))
	if err != nil {
		t.Fatalf("add returned go error: %v", err)
	}
	if !result.IsError {
		t.Fatal("expected a tool error when inbound_ids is missing")
	}
}

// TestUpdateClient_PreservesOmittedFields verifies that update reads the current
// client and only overlays supplied fields (so the UUID is not regenerated).
func TestUpdateClient_PreservesOmittedFields(t *testing.T) {
	var updateBody xui.ClientConfig

	h, _ := newClientHandler(t, func(w http.ResponseWriter, r *http.Request) {
		if writeCSRF(w, r) {
			return
		}
		switch r.URL.Path {
		case "/login":
			json.NewEncoder(w).Encode(map[string]any{"success": true})
		case "/panel/api/clients/get/user@example.com":
			json.NewEncoder(w).Encode(map[string]any{
				"success": true,
				"obj": map[string]any{
					"client": map[string]any{
						"uuid":    "keep-this-uuid",
						"email":   "user@example.com",
						"totalGB": 1073741824,
						"enable":  true,
						"subId":   "sub-1",
					},
					"inboundIds": []int{2},
				},
			})
		case "/panel/api/clients/update/user@example.com":
			if err := json.NewDecoder(r.Body).Decode(&updateBody); err != nil {
				t.Errorf("decoding update body: %v", err)
			}
			json.NewEncoder(w).Encode(map[string]any{"success": true, "msg": "updated"})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	})

	// Only change the traffic limit; everything else must be preserved.
	result, err := h.update(context.Background(), req(map[string]any{
		"email":    "user@example.com",
		"total_gb": float64(10),
	}))
	if err != nil {
		t.Fatalf("update returned go error: %v", err)
	}
	if result.IsError {
		t.Fatalf("update returned tool error: %+v", result.Content)
	}

	if updateBody.ID != "keep-this-uuid" {
		t.Errorf("UUID = %q, want it preserved as %q", updateBody.ID, "keep-this-uuid")
	}
	if updateBody.SubID != "sub-1" {
		t.Errorf("subId = %q, want preserved %q", updateBody.SubID, "sub-1")
	}
	if updateBody.TotalGB != 10*1073741824 {
		t.Errorf("totalGB = %d, want %d (10 GB)", updateBody.TotalGB, 10*1073741824)
	}
}
