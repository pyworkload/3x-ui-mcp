package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
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

// add_gb is the tool's unit; the panel's is bytes. A fractional GB has to
// survive the conversion, since renewals are often sold in half-terabytes.
func TestBulkAdjustClients_ConvertsGBToBytes(t *testing.T) {
	var gotBody map[string]any

	h, _ := newClientHandler(t, func(w http.ResponseWriter, r *http.Request) {
		if writeCSRF(w, r) {
			return
		}
		switch r.URL.Path {
		case "/login":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
		case "/panel/api/clients/bulkAdjust":
			if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
				t.Errorf("decoding body: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	})

	res, err := h.bulkAdjust(context.Background(), req(map[string]any{
		"emails": []any{"alice"},
		"add_gb": 1.5,
	}))
	if err != nil {
		t.Fatalf("bulkAdjust returned error: %v", err)
	}
	if res.IsError {
		t.Fatalf("bulkAdjust reported a tool error: %+v", res.Content)
	}

	const wantBytes = 1.5 * bytesPerGB
	if gotBody["addBytes"] != float64(wantBytes) {
		t.Errorf("addBytes = %v, want %v", gotBody["addBytes"], float64(wantBytes))
	}
}

// Every delta defaulting to zero would send the panel a no-op that still
// reports success, so the tool asks for one before making the call.
func TestBulkAdjustClients_RejectsEmptyAdjustment(t *testing.T) {
	h, _ := newClientHandler(t, func(w http.ResponseWriter, r *http.Request) {
		if writeCSRF(w, r) {
			return
		}
		if r.URL.Path == "/login" {
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
			return
		}
		t.Errorf("panel was called for an empty adjustment: %s", r.URL.Path)
	})

	res, err := h.bulkAdjust(context.Background(), req(map[string]any{"emails": []any{"alice"}}))
	if err != nil {
		t.Fatalf("bulkAdjust returned error: %v", err)
	}
	if !res.IsError {
		t.Error("bulkAdjust accepted an adjustment with nothing to adjust")
	}
}

// The paged listing builds a query string, and empty filters must not become
// empty query keys — the panel treats an empty filter as a real bucket name.
func TestListClientsPaged_OmitsEmptyFilters(t *testing.T) {
	var gotQuery string

	h, _ := newClientHandler(t, func(w http.ResponseWriter, r *http.Request) {
		if writeCSRF(w, r) {
			return
		}
		if r.URL.Path == "/login" {
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
			return
		}
		gotQuery = r.URL.RawQuery
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "obj": map[string]any{"items": []any{}}})
	})

	res, err := h.listPaged(context.Background(), req(map[string]any{
		"page_size": float64(50),
		"sort":      "traffic",
	}))
	if err != nil {
		t.Fatalf("listPaged returned error: %v", err)
	}
	if res.IsError {
		t.Fatalf("listPaged reported a tool error: %+v", res.Content)
	}

	if !strings.Contains(gotQuery, "pageSize=50") || !strings.Contains(gotQuery, "sort=traffic") {
		t.Errorf("query = %q, want pageSize and sort carried through", gotQuery)
	}
	for _, unwanted := range []string{"search=", "filter=", "protocol=", "order="} {
		if strings.Contains(gotQuery, unwanted) {
			t.Errorf("query = %q, want %q omitted when unset", gotQuery, unwanted)
		}
	}
}
