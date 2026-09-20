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

// sampleClientRecord is the "client" object a live v3.8.5 panel returns from
// GET clients/get/:email. The panel answers with a ClientRecord and reads a
// Client back: the UUID arrives under "uuid" but is written as "id", the
// WireGuard addresses arrive comma-separated but are written as an array, and
// the row id and timestamps are the panel's own.
const sampleClientRecord = `{
	"id": 43,
	"uuid": "keep-this-uuid",
	"email": "user@example.com",
	"subId": "sub-1",
	"password": "keep-this-password",
	"auth": "keep-this-auth",
	"secret": "keep-this-secret",
	"flow": "xtls-rprx-vision",
	"security": "auto",
	"reverse": {"tag": "reverse-tag"},
	"privateKey": "wg-private",
	"publicKey": "wg-public",
	"allowedIPs": "10.0.0.2/32, fd00::2/128",
	"preSharedKey": "wg-psk",
	"keepAlive": 25,
	"forwardedPorts": "80,443",
	"adTag": "ad-tag",
	"limitIp": 3,
	"limitHwid": 2,
	"totalGB": 1073741824,
	"expiryTime": 0,
	"enable": true,
	"tgId": 0,
	"group": "vip",
	"comment": "note",
	"reset": 0,
	"resetDay": 7,
	"resetMax": 4,
	"trafficReset": "monthly",
	"trafficResetDay": 14,
	"createdAt": 1784657021231,
	"updatedAt": 1789840286000
}`

// updateClientHandler serves the record above and captures the update body.
func updateClientHandler(t *testing.T) (*clientHandler, *map[string]any) {
	t.Helper()
	var body map[string]any
	h, _ := newClientHandler(t, func(w http.ResponseWriter, r *http.Request) {
		if writeCSRF(w, r) {
			return
		}
		switch r.URL.Path {
		case "/login":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
		case "/panel/api/clients/get/user@example.com":
			_, _ = w.Write([]byte(`{"success":true,"obj":{"client":` + sampleClientRecord + `,"inboundIds":[2]}}`))
		case "/panel/api/clients/update/user@example.com":
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Errorf("decoding update body: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "msg": "updated"})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	})
	return h, &body
}

// TestUpdateClient_PreservesOmittedFields verifies that update reads the current
// client and only overlays supplied fields (so the UUID is not regenerated).
func TestUpdateClient_PreservesOmittedFields(t *testing.T) {
	h, body := updateClientHandler(t)

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

	got := *body
	if got["id"] != "keep-this-uuid" {
		t.Errorf("id = %v, want the record's uuid preserved as %q", got["id"], "keep-this-uuid")
	}
	if got["subId"] != "sub-1" {
		t.Errorf("subId = %v, want preserved %q", got["subId"], "sub-1")
	}
	if got["totalGB"] != float64(10*1073741824) {
		t.Errorf("totalGB = %v, want %d (10 GB)", got["totalGB"], 10*1073741824)
	}
}

// The update endpoint replaces the client outright and writes reverse, adTag
// and limitHwid unconditionally, so a field the tool has no parameter for is
// cleared unless the current value is sent back with it.
func TestUpdateClient_PreservesFieldsWithoutAToolParameter(t *testing.T) {
	h, body := updateClientHandler(t)

	result, err := h.update(context.Background(), req(map[string]any{
		"email":   "user@example.com",
		"comment": "renewed",
	}))
	if err != nil {
		t.Fatalf("update returned go error: %v", err)
	}
	if result.IsError {
		t.Fatalf("update returned tool error: %+v", result.Content)
	}

	got := *body
	if got["comment"] != "renewed" {
		t.Errorf("comment = %v, want %q", got["comment"], "renewed")
	}
	for key, want := range map[string]any{
		"privateKey":      "wg-private",
		"publicKey":       "wg-public",
		"preSharedKey":    "wg-psk",
		"keepAlive":       float64(25),
		"forwardedPorts":  "80,443",
		"adTag":           "ad-tag",
		"secret":          "keep-this-secret",
		"auth":            "keep-this-auth",
		"limitHwid":       float64(2),
		"resetDay":        float64(7),
		"resetMax":        float64(4),
		"trafficReset":    "monthly",
		"trafficResetDay": float64(14),
		"group":           "vip",
		"flow":            "xtls-rprx-vision",
	} {
		if got[key] != want {
			t.Errorf("%s = %v, want %v (the panel clears whatever it is not sent)", key, got[key], want)
		}
	}

	reverse, ok := got["reverse"].(map[string]any)
	if !ok || reverse["tag"] != "reverse-tag" {
		t.Errorf("reverse = %v, want the stored {tag: reverse-tag} (the panel blanks the column otherwise)", got["reverse"])
	}
}

// The record spells the UUID and the WireGuard addresses differently from the
// body: "uuid" vs "id", and a comma-separated string vs an array. The row id
// and the timestamps are the panel's and must not be echoed at all — "id" in
// the body means the UUID, so sending the row's integer id there fails to bind.
func TestUpdateClient_TranslatesRecordShapeIntoBodyShape(t *testing.T) {
	h, body := updateClientHandler(t)

	result, err := h.update(context.Background(), req(map[string]any{
		"email":  "user@example.com",
		"enable": false,
	}))
	if err != nil {
		t.Fatalf("update returned go error: %v", err)
	}
	if result.IsError {
		t.Fatalf("update returned tool error: %+v", result.Content)
	}

	got := *body
	if got["enable"] != false {
		t.Errorf("enable = %v, want false", got["enable"])
	}
	if _, ok := got["uuid"]; ok {
		t.Error("uuid leaked into the body — the panel reads the UUID from id")
	}
	for _, key := range []string{"createdAt", "updatedAt"} {
		if _, ok := got[key]; ok {
			t.Errorf("%s leaked into the body — the panel maintains it itself", key)
		}
	}

	ips, ok := got["allowedIPs"].([]any)
	if !ok {
		t.Fatalf("allowedIPs = %#v, want an array (the body's AllowedIPs is []string)", got["allowedIPs"])
	}
	if len(ips) != 2 || ips[0] != "10.0.0.2/32" || ips[1] != "fd00::2/128" {
		t.Errorf("allowedIPs = %v, want the two addresses split and trimmed", ips)
	}
}

// The fields above were preserved long before they could be set. Now that each
// one has a parameter, the overlay has to land it in the shape the body uses:
// limitHwid at the top level, reverse as an object, allowedIPs as an array.
func TestUpdateClient_OverlaysTheProtocolFields(t *testing.T) {
	h, body := updateClientHandler(t)

	result, err := h.update(context.Background(), req(map[string]any{
		"email":             "user@example.com",
		"limit_hwid":        float64(5),
		"reset_day":         float64(1),
		"reset_max":         float64(0),
		"traffic_reset":     "weekly",
		"traffic_reset_day": float64(3),
		"reverse_tag":       "new-reverse",
		"secret":            "new-secret",
		"ad_tag":            strings.Repeat("a", 32),
		"private_key":       "new-private",
		"public_key":        "new-public",
		"pre_shared_key":    "new-psk",
		"forwarded_ports":   "80,443,8000-8100",
		"keep_alive":        float64(0),
		"allowed_ips":       []any{"10.0.0.9/32"},
	}))
	if err != nil {
		t.Fatalf("update returned go error: %v", err)
	}
	if result.IsError {
		t.Fatalf("update returned tool error: %+v", result.Content)
	}

	got := *body
	for key, want := range map[string]any{
		"limitHwid":       float64(5),
		"resetDay":        float64(1),
		"resetMax":        float64(0),
		"trafficReset":    "weekly",
		"trafficResetDay": float64(3),
		"secret":          "new-secret",
		"adTag":           strings.Repeat("a", 32),
		"privateKey":      "new-private",
		"publicKey":       "new-public",
		"preSharedKey":    "new-psk",
		"forwardedPorts":  "80,443,8000-8100",
		// An explicit 0 means "send no keepalive packets" and must not be
		// mistaken for the parameter being absent.
		"keepAlive": float64(0),
	} {
		if got[key] != want {
			t.Errorf("%s = %v, want %v", key, got[key], want)
		}
	}

	reverse, ok := got["reverse"].(map[string]any)
	if !ok || reverse["tag"] != "new-reverse" {
		t.Errorf("reverse = %v, want {tag: new-reverse}", got["reverse"])
	}
	ips, ok := got["allowedIPs"].([]any)
	if !ok || len(ips) != 1 || ips[0] != "10.0.0.9/32" {
		t.Errorf("allowedIPs = %v, want [10.0.0.9/32]", got["allowedIPs"])
	}
}

// An empty reverse_tag is the only way to drop the reverse proxy: sending
// {"tag": ""} would leave the client pointing at a tag that routes nowhere.
func TestUpdateClient_EmptyReverseTagClearsIt(t *testing.T) {
	h, body := updateClientHandler(t)

	result, err := h.update(context.Background(), req(map[string]any{
		"email":       "user@example.com",
		"reverse_tag": "",
	}))
	if err != nil {
		t.Fatalf("update returned go error: %v", err)
	}
	if result.IsError {
		t.Fatalf("update returned tool error: %+v", result.Content)
	}
	if _, ok := (*body)["reverse"]; ok {
		t.Errorf("reverse = %v, want it dropped from the body", (*body)["reverse"])
	}
}

// clients/add reads limitHwid from inside the client object, while
// clients/update reads it from the top level of the body.
func TestAddClient_CarriesTheProtocolFields(t *testing.T) {
	var raw map[string]any

	h, _ := newClientHandler(t, func(w http.ResponseWriter, r *http.Request) {
		if writeCSRF(w, r) {
			return
		}
		switch r.URL.Path {
		case "/login":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
		case "/panel/api/clients/add":
			if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
				t.Errorf("decoding payload: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "msg": "added"})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	})

	result, err := h.add(context.Background(), req(map[string]any{
		"inbound_ids":   []any{float64(1)},
		"email":         "user@example.com",
		"limit_hwid":    float64(4),
		"traffic_reset": "daily",
		"reverse_tag":   "rev",
		"allowed_ips":   []any{"10.0.0.2/32"},
		"keep_alive":    float64(0),
	}))
	if err != nil {
		t.Fatalf("add returned go error: %v", err)
	}
	if result.IsError {
		t.Fatalf("add returned tool error: %+v", result.Content)
	}

	client, ok := raw["client"].(map[string]any)
	if !ok {
		t.Fatalf("payload = %#v, want a {client, inboundIds} object", raw)
	}
	if client["limitHwid"] != float64(4) {
		t.Errorf("client.limitHwid = %v, want 4 (add reads it from inside the client)", client["limitHwid"])
	}
	if client["trafficReset"] != "daily" {
		t.Errorf("client.trafficReset = %v, want %q", client["trafficReset"], "daily")
	}
	reverse, ok := client["reverse"].(map[string]any)
	if !ok || reverse["tag"] != "rev" {
		t.Errorf("client.reverse = %v, want {tag: rev}", client["reverse"])
	}
	if client["keepAlive"] != float64(0) {
		t.Errorf("client.keepAlive = %v, want an explicit 0", client["keepAlive"])
	}
}

// Nothing the caller left out may reach the panel: an omitted keep_alive is
// not the same as 0, and an empty reverse_tag is not an empty reverse object.
func TestAddClient_OmitsUnsetProtocolFields(t *testing.T) {
	var raw map[string]any

	h, _ := newClientHandler(t, func(w http.ResponseWriter, r *http.Request) {
		if writeCSRF(w, r) {
			return
		}
		switch r.URL.Path {
		case "/login":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
		case "/panel/api/clients/add":
			if err := json.NewDecoder(r.Body).Decode(&raw); err != nil {
				t.Errorf("decoding payload: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "msg": "added"})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	})

	if _, err := h.add(context.Background(), req(map[string]any{
		"inbound_ids": []any{float64(1)},
		"email":       "user@example.com",
	})); err != nil {
		t.Fatalf("add returned go error: %v", err)
	}

	client, ok := raw["client"].(map[string]any)
	if !ok {
		t.Fatalf("payload = %#v, want a {client, inboundIds} object", raw)
	}
	for _, key := range []string{"keepAlive", "reverse", "limitHwid", "trafficReset", "allowedIPs", "adTag", "secret", "privateKey"} {
		if _, ok := client[key]; ok {
			t.Errorf("%s = %v, want it left out so the panel applies its own default", key, client[key])
		}
	}
}

func TestClientBaseFromRecord_DropsEmptyAllowedIPs(t *testing.T) {
	base, err := clientBaseFromRecord(json.RawMessage(`{"client":{"uuid":"u","email":"e","allowedIPs":""}}`))
	if err != nil {
		t.Fatalf("clientBaseFromRecord: %v", err)
	}
	if _, ok := base["allowedIPs"]; ok {
		t.Error("an empty address list should be omitted, which the panel reads as keep-the-stored-one")
	}
}

func TestClientBaseFromRecord_RejectsEmptyRecord(t *testing.T) {
	if _, err := clientBaseFromRecord(json.RawMessage(`{"client":{}}`)); err == nil {
		t.Error("expected an error for a record with no fields")
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
