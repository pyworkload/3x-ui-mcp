package xui

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// recordedRequest is what the panel saw for a single client call.
type recordedRequest struct {
	path        string
	escapedPath string
	method      string
	contentType string
	body        string
}

// recordingClient wires a Client to a server that answers every API call with
// a bare success and records what it was sent.
func recordingClient(t *testing.T) (*Client, *recordedRequest) {
	t.Helper()
	rec := &recordedRequest{}
	_, client := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if writeCSRFIfRequested(w, r) {
			return
		}
		if r.URL.Path == "/login" {
			_ = json.NewEncoder(w).Encode(Response{Success: true})
			return
		}
		body, _ := io.ReadAll(r.Body)
		rec.path = r.URL.Path
		rec.escapedPath = r.URL.EscapedPath()
		rec.method = r.Method
		rec.contentType = r.Header.Get("Content-Type")
		rec.body = string(body)
		_ = json.NewEncoder(w).Encode(Response{Success: true, Obj: json.RawMessage(`{}`)})
	})
	return client, rec
}

func TestSetInboundEnable_PostsFlagToIDPath(t *testing.T) {
	client, rec := recordingClient(t)

	if _, err := client.SetInboundEnable(context.Background(), 7, false); err != nil {
		t.Fatalf("SetInboundEnable returned error: %v", err)
	}

	if rec.path != "/panel/api/inbounds/setEnable/7" {
		t.Errorf("path = %q, want %q", rec.path, "/panel/api/inbounds/setEnable/7")
	}
	if rec.body != "enable=false" {
		t.Errorf("body = %q, want %q", rec.body, "enable=false")
	}
}

func TestResetInboundTraffic_PostsToIDPath(t *testing.T) {
	client, rec := recordingClient(t)

	if _, err := client.ResetInboundTraffic(context.Background(), 12); err != nil {
		t.Fatalf("ResetInboundTraffic returned error: %v", err)
	}

	if rec.path != "/panel/api/inbounds/12/resetTraffic" {
		t.Errorf("path = %q, want %q", rec.path, "/panel/api/inbounds/12/resetTraffic")
	}
	if rec.method != http.MethodPost {
		t.Errorf("method = %q, want POST", rec.method)
	}
}

func TestDeleteAllInboundClients_PostsToIDPath(t *testing.T) {
	client, rec := recordingClient(t)

	if _, err := client.DeleteAllInboundClients(context.Background(), 3); err != nil {
		t.Fatalf("DeleteAllInboundClients returned error: %v", err)
	}

	if rec.path != "/panel/api/inbounds/3/delAllClients" {
		t.Errorf("path = %q, want %q", rec.path, "/panel/api/inbounds/3/delAllClients")
	}
}

func TestBulkDeleteInbounds_SendsJSONIDs(t *testing.T) {
	client, rec := recordingClient(t)

	if _, err := client.BulkDeleteInbounds(context.Background(), []int{3, 7, 12}); err != nil {
		t.Fatalf("BulkDeleteInbounds returned error: %v", err)
	}

	if rec.path != "/panel/api/inbounds/bulkDel" {
		t.Errorf("path = %q, want %q", rec.path, "/panel/api/inbounds/bulkDel")
	}
	if !strings.Contains(rec.contentType, "application/json") {
		t.Errorf("content type = %q, want JSON", rec.contentType)
	}
	var got struct {
		Ids []int `json:"ids"`
	}
	if err := json.Unmarshal([]byte(rec.body), &got); err != nil {
		t.Fatalf("body is not JSON: %v (%q)", err, rec.body)
	}
	if len(got.Ids) != 3 || got.Ids[0] != 3 || got.Ids[2] != 12 {
		t.Errorf("ids = %v, want [3 7 12]", got.Ids)
	}
}

func TestGetBalancersStatus_JoinsTags(t *testing.T) {
	client, rec := recordingClient(t)

	if _, err := client.GetBalancersStatus(context.Background(), []string{"LB-A", "LB-B"}); err != nil {
		t.Fatalf("GetBalancersStatus returned error: %v", err)
	}

	if rec.path != "/panel/api/xray/balancerStatus" {
		t.Errorf("path = %q, want %q", rec.path, "/panel/api/xray/balancerStatus")
	}
	if rec.body != "tags=LB-A%2CLB-B" {
		t.Errorf("body = %q, want the tags comma-joined", rec.body)
	}
}

func TestOverrideBalancer_SendsTagAndTarget(t *testing.T) {
	client, rec := recordingClient(t)

	if _, err := client.OverrideBalancer(context.Background(), "LB-PROXY", "out-nl"); err != nil {
		t.Fatalf("OverrideBalancer returned error: %v", err)
	}

	if rec.path != "/panel/api/xray/balancerOverride" {
		t.Errorf("path = %q, want %q", rec.path, "/panel/api/xray/balancerOverride")
	}
	if !strings.Contains(rec.body, "tag=LB-PROXY") || !strings.Contains(rec.body, "target=out-nl") {
		t.Errorf("body = %q, want both tag and target", rec.body)
	}
}

// An empty target is how the panel clears an override, so the field has to go
// out on the wire rather than being dropped as "unset".
func TestOverrideBalancer_EmptyTargetStillSent(t *testing.T) {
	client, rec := recordingClient(t)

	if _, err := client.OverrideBalancer(context.Background(), "LB-PROXY", ""); err != nil {
		t.Fatalf("OverrideBalancer returned error: %v", err)
	}

	if !strings.Contains(rec.body, "target=") {
		t.Errorf("body = %q, want an explicit empty target", rec.body)
	}
}

func TestTestRoute_SendsOnlySetFields(t *testing.T) {
	client, rec := recordingClient(t)

	_, err := client.TestRoute(context.Background(), RouteTestRequest{
		Domain:  "www.youtube.com",
		Port:    443,
		Network: "tcp",
	})
	if err != nil {
		t.Fatalf("TestRoute returned error: %v", err)
	}

	if rec.path != "/panel/api/xray/routeTest" {
		t.Errorf("path = %q, want %q", rec.path, "/panel/api/xray/routeTest")
	}
	for _, want := range []string{"domain=www.youtube.com", "port=443", "network=tcp"} {
		if !strings.Contains(rec.body, want) {
			t.Errorf("body = %q, want it to contain %q", rec.body, want)
		}
	}
	for _, unwanted := range []string{"ip=", "email=", "protocol=", "inboundTag="} {
		if strings.Contains(rec.body, unwanted) {
			t.Errorf("body = %q, should not carry unset field %q", rec.body, unwanted)
		}
	}
}

// xver=0 means "no PROXY protocol"; sending it would make the panel probe with
// a version it never asked for.
func TestScanRealityTarget_OmitsZeroXver(t *testing.T) {
	client, rec := recordingClient(t)

	if _, err := client.ScanRealityTarget(context.Background(), "www.cloudflare.com", 0); err != nil {
		t.Fatalf("ScanRealityTarget returned error: %v", err)
	}

	if rec.body != "target=www.cloudflare.com" {
		t.Errorf("body = %q, want only the target", rec.body)
	}
}

func TestScanRealityTarget_SendsXver(t *testing.T) {
	client, rec := recordingClient(t)

	if _, err := client.ScanRealityTarget(context.Background(), "www.cloudflare.com", 2); err != nil {
		t.Fatalf("ScanRealityTarget returned error: %v", err)
	}

	if !strings.Contains(rec.body, "xver=2") {
		t.Errorf("body = %q, want xver=2", rec.body)
	}
}

func TestKeyGenerators_HitTheirEndpoints(t *testing.T) {
	tests := []struct {
		name string
		call func(*Client, context.Context) (*Response, error)
		path string
	}{
		{"uuid", (*Client).GetNewUUID, "/panel/api/server/getNewUUID"},
		{"x25519", (*Client).GetNewX25519Cert, "/panel/api/server/getNewX25519Cert"},
		{"vless_enc", (*Client).GetNewVlessEnc, "/panel/api/server/getNewVlessEnc"},
		{"mlkem768", (*Client).GetNewMLKEM768, "/panel/api/server/getNewmlkem768"},
		{"mldsa65", (*Client).GetNewMLDSA65, "/panel/api/server/getNewmldsa65"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, rec := recordingClient(t)

			if _, err := tt.call(client, context.Background()); err != nil {
				t.Fatalf("%s returned error: %v", tt.name, err)
			}

			if rec.path != tt.path {
				t.Errorf("path = %q, want %q", rec.path, tt.path)
			}
			if rec.method != http.MethodGet {
				t.Errorf("method = %q, want GET", rec.method)
			}
		})
	}
}

func TestCreateOutboundSub_SendsAllFields(t *testing.T) {
	client, rec := recordingClient(t)

	_, err := client.CreateOutboundSub(context.Background(), OutboundSubParams{
		Remark:         "nl-pool",
		URL:            "https://example.com/sub",
		TagPrefix:      "nl-",
		Enabled:        true,
		UpdateInterval: 900,
		AllowPrivate:   false,
		AllowInsecure:  true,
		Prepend:        true,
	})
	if err != nil {
		t.Fatalf("CreateOutboundSub returned error: %v", err)
	}

	if rec.path != "/panel/api/xray/outbound-subs" {
		t.Errorf("path = %q, want %q", rec.path, "/panel/api/xray/outbound-subs")
	}
	form, err := url.ParseQuery(rec.body)
	if err != nil {
		t.Fatalf("body is not a form: %v", err)
	}
	want := map[string]string{
		"remark":         "nl-pool",
		"url":            "https://example.com/sub",
		"tagPrefix":      "nl-",
		"enabled":        "true",
		"updateInterval": "900",
		"allowPrivate":   "false",
		"allowInsecure":  "true",
		"prepend":        "true",
	}
	for field, value := range want {
		if form.Get(field) != value {
			t.Errorf("%s = %q, want %q", field, form.Get(field), value)
		}
	}
}

// The panel reads "enabled" as anything-but-"false", so a disabled
// subscription depends on the literal string going out.
func TestCreateOutboundSub_DisabledSendsFalse(t *testing.T) {
	client, rec := recordingClient(t)

	if _, err := client.CreateOutboundSub(context.Background(), OutboundSubParams{URL: "u", Enabled: false}); err != nil {
		t.Fatalf("CreateOutboundSub returned error: %v", err)
	}

	form, _ := url.ParseQuery(rec.body)
	if form.Get("enabled") != "false" {
		t.Errorf("enabled = %q, want %q", form.Get("enabled"), "false")
	}
}

func TestUpdateOutboundSub_PostsToIDPath(t *testing.T) {
	client, rec := recordingClient(t)

	if _, err := client.UpdateOutboundSub(context.Background(), 4, OutboundSubParams{URL: "u", Enabled: true}); err != nil {
		t.Fatalf("UpdateOutboundSub returned error: %v", err)
	}

	if rec.path != "/panel/api/xray/outbound-subs/4" {
		t.Errorf("path = %q, want %q", rec.path, "/panel/api/xray/outbound-subs/4")
	}
}

func TestDeleteOutboundSub_UsesPostAlias(t *testing.T) {
	client, rec := recordingClient(t)

	if _, err := client.DeleteOutboundSub(context.Background(), 4); err != nil {
		t.Fatalf("DeleteOutboundSub returned error: %v", err)
	}

	if rec.path != "/panel/api/xray/outbound-subs/4/del" {
		t.Errorf("path = %q, want %q", rec.path, "/panel/api/xray/outbound-subs/4/del")
	}
	if rec.method != http.MethodPost {
		t.Errorf("method = %q, want POST", rec.method)
	}
}

func TestMoveOutboundSub_SendsDirection(t *testing.T) {
	tests := []struct {
		name string
		up   bool
		want string
	}{
		{"up", true, "dir=up"},
		{"down", false, "dir=down"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, rec := recordingClient(t)

			if _, err := client.MoveOutboundSub(context.Background(), 4, tt.up); err != nil {
				t.Fatalf("MoveOutboundSub returned error: %v", err)
			}

			if rec.path != "/panel/api/xray/outbound-subs/4/move" {
				t.Errorf("path = %q, want the move path", rec.path)
			}
			if rec.body != tt.want {
				t.Errorf("body = %q, want %q", rec.body, tt.want)
			}
		})
	}
}

func TestGetSubscriptionLinks_EscapesSubID(t *testing.T) {
	client, rec := recordingClient(t)

	if _, err := client.GetSubscriptionLinks(context.Background(), "team a/1"); err != nil {
		t.Fatalf("GetSubscriptionLinks returned error: %v", err)
	}

	if rec.escapedPath != "/panel/api/clients/subLinks/team%20a%2F1" {
		t.Errorf("escaped path = %q, want the sub id path-escaped", rec.escapedPath)
	}
	if rec.path != "/panel/api/clients/subLinks/team a/1" {
		t.Errorf("decoded path = %q, want the sub id intact", rec.path)
	}
}
