package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/pyworkload/3x-ui-mcp/internal/config"
	"github.com/pyworkload/3x-ui-mcp/internal/xui"

	"github.com/mark3labs/mcp-go/mcp"
)

// resultText returns the text a tool result carries, failing the test if it
// carries something else.
func resultText(t *testing.T, result *mcp.CallToolResult) string {
	t.Helper()
	if len(result.Content) == 0 {
		t.Fatal("tool result has no content")
	}
	text, ok := result.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("content is %T, want text", result.Content[0])
	}
	return text.Text
}

// templateWithBalancers mirrors the shape the panel returns: routing carries
// both rules and balancers, and a rule points at a balancer by tag.
const templateWithBalancers = `{
	"outbounds": [{"tag": "out-nl"}, {"tag": "out-de"}],
	"routing": {
		"domainStrategy": "IPIfNonMatch",
		"rules": [{"type": "field", "network": "tcp,udp", "balancerTag": "LB-PROXY"}],
		"balancers": [
			{"tag": "LB-PROXY", "selector": ["out-nl", "out-de"], "strategy": {"type": "leastLoad"}},
			{"tag": "LB-IDLE", "selector": ["out-de"]}
		]
	}
}`

func parseTemplate(t *testing.T, raw string) map[string]any {
	t.Helper()
	var template map[string]any
	if err := json.Unmarshal([]byte(raw), &template); err != nil {
		t.Fatalf("test template is not valid JSON: %v", err)
	}
	return template
}

func TestGetBalancersFromTemplate_ReturnsDefinitions(t *testing.T) {
	balancers := getBalancersFromTemplate(parseTemplate(t, templateWithBalancers))

	if len(balancers) != 2 {
		t.Fatalf("got %d balancers, want 2", len(balancers))
	}
	first, ok := balancers[0].(map[string]any)
	if !ok {
		t.Fatalf("balancer 0 is %T, want an object", balancers[0])
	}
	if first["tag"] != "LB-PROXY" {
		t.Errorf("tag = %v, want LB-PROXY", first["tag"])
	}
}

func TestGetBalancersFromTemplate_MissingSections(t *testing.T) {
	tests := []struct {
		name     string
		template string
	}{
		{"no routing", `{"outbounds": []}`},
		{"routing without balancers", `{"routing": {"rules": []}}`},
		{"balancers not an array", `{"routing": {"balancers": {"tag": "x"}}}`},
		{"routing not an object", `{"routing": "nope"}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getBalancersFromTemplate(parseTemplate(t, tt.template)); got != nil {
				t.Errorf("got %v, want nil", got)
			}
		})
	}
}

func TestBalancerTags_SkipsEntriesWithoutTag(t *testing.T) {
	balancers := []any{
		map[string]any{"tag": "LB-A"},
		map[string]any{"selector": []any{"out"}}, // no tag
		map[string]any{"tag": ""},                // empty tag
		"not an object",
		map[string]any{"tag": "LB-B"},
	}

	tags := balancerTags(balancers)

	if len(tags) != 2 || tags[0] != "LB-A" || tags[1] != "LB-B" {
		t.Errorf("tags = %v, want [LB-A LB-B]", tags)
	}
}

func TestMergeBalancerStatus_AttachesLiveState(t *testing.T) {
	balancers := getBalancersFromTemplate(parseTemplate(t, templateWithBalancers))
	status := map[string]any{
		"LB-PROXY": map[string]any{
			"tag":      "LB-PROXY",
			"running":  true,
			"override": "",
			"selected": []any{"out-de"},
		},
	}

	merged := mergeBalancerStatus(balancers, status)

	if len(merged) != 2 {
		t.Fatalf("got %d entries, want 2", len(merged))
	}

	withStatus := merged[0].(map[string]any)
	if withStatus["tag"] != "LB-PROXY" {
		t.Errorf("tag = %v, want LB-PROXY", withStatus["tag"])
	}
	if withStatus["selector"] == nil {
		t.Error("expected the saved definition to survive the merge")
	}
	live, ok := withStatus["live"].(map[string]any)
	if !ok {
		t.Fatalf("live = %T, want an object", withStatus["live"])
	}
	if live["running"] != true {
		t.Errorf("live.running = %v, want true", live["running"])
	}

	// A balancer the core doesn't report still shows its saved config.
	withoutStatus := merged[1].(map[string]any)
	if _, hasLive := withoutStatus["live"]; hasLive {
		t.Error("expected no live state for a balancer missing from the status map")
	}
	if withoutStatus["tag"] != "LB-IDLE" {
		t.Errorf("tag = %v, want LB-IDLE", withoutStatus["tag"])
	}
}

// The merge must not write into the template it was handed — the same map is
// reused by callers that go on to save the template.
func TestMergeBalancerStatus_DoesNotMutateInput(t *testing.T) {
	balancers := getBalancersFromTemplate(parseTemplate(t, templateWithBalancers))
	status := map[string]any{"LB-PROXY": map[string]any{"running": true}}

	mergeBalancerStatus(balancers, status)

	original := balancers[0].(map[string]any)
	if _, polluted := original["live"]; polluted {
		t.Error("merge leaked live state into the source balancer definition")
	}
}

func newXrayHandler(t *testing.T, handler http.HandlerFunc) *xrayHandler {
	t.Helper()
	ts := httptest.NewServer(handler)
	t.Cleanup(ts.Close)
	cfg := &config.Config{Host: ts.URL, BasePath: "/", Username: "admin", Password: "admin"}
	return &xrayHandler{client: xui.NewClient(cfg, slog.Default())}
}

// panelWithTemplate answers the login dance plus a template fetch, and hands
// anything else to extra.
func panelWithTemplate(t *testing.T, template string, extra http.HandlerFunc) http.HandlerFunc {
	t.Helper()
	return func(w http.ResponseWriter, r *http.Request) {
		if writeCSRF(w, r) {
			return
		}
		switch r.URL.Path {
		case "/login":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
		case "/panel/api/xray/":
			// The panel wraps the template as a JSON string inside the response.
			wrapped, _ := json.Marshal(map[string]any{"xraySetting": template})
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "obj": string(wrapped)})
		default:
			extra(w, r)
		}
	}
}

func TestGetBalancers_MergesLiveState(t *testing.T) {
	h := newXrayHandler(t, panelWithTemplate(t, templateWithBalancers, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/panel/api/xray/balancerStatus" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			return
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parsing form: %v", err)
		}
		if got := r.PostForm.Get("tags"); got != "LB-PROXY,LB-IDLE" {
			t.Errorf("tags = %q, want both balancer tags", got)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "obj": map[string]any{
			"LB-PROXY": map[string]any{"tag": "LB-PROXY", "running": true, "selected": []string{"out-de"}},
		}})
	}))

	result, err := h.getBalancers(context.Background(), req(map[string]any{}))
	if err != nil {
		t.Fatalf("getBalancers returned error: %v", err)
	}
	text := resultText(t, result)
	for _, want := range []string{`"live"`, `"running": true`, "out-de", "LB-IDLE"} {
		if !strings.Contains(text, want) {
			t.Errorf("output missing %q:\n%s", want, text)
		}
	}
}

// A core that can't answer must not cost the caller the saved definitions.
func TestGetBalancers_ReportsStatusFailureButKeepsConfig(t *testing.T) {
	h := newXrayHandler(t, panelWithTemplate(t, templateWithBalancers, func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"success": false, "msg": "xray is not running"})
	}))

	result, err := h.getBalancers(context.Background(), req(map[string]any{}))
	if err != nil {
		t.Fatalf("getBalancers returned error: %v", err)
	}
	text := resultText(t, result)
	if !strings.Contains(text, "live_status_error") || !strings.Contains(text, "xray is not running") {
		t.Errorf("expected the status failure to be reported:\n%s", text)
	}
	if !strings.Contains(text, "LB-PROXY") {
		t.Errorf("expected the saved definitions to survive:\n%s", text)
	}
}

func TestGetBalancers_NoBalancers(t *testing.T) {
	h := newXrayHandler(t, panelWithTemplate(t, `{"routing":{"rules":[]}}`, func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("balancer status should not be queried when there are none (path %s)", r.URL.Path)
	}))

	result, err := h.getBalancers(context.Background(), req(map[string]any{}))
	if err != nil {
		t.Fatalf("getBalancers returned error: %v", err)
	}
	if !strings.Contains(resultText(t, result), "No balancers") {
		t.Errorf("unexpected output: %s", resultText(t, result))
	}
}

const storedOutboundSub = `[{
	"id": 4,
	"remark": "nl-pool",
	"url": "https://example.com/sub",
	"tagPrefix": "nl-",
	"enabled": true,
	"updateInterval": 900,
	"allowPrivate": false,
	"allowInsecure": true,
	"prepend": true
}]`

func TestUpdateOutboundSub_PreservesOmittedFields(t *testing.T) {
	var got url.Values

	h := newXrayHandler(t, func(w http.ResponseWriter, r *http.Request) {
		if writeCSRF(w, r) {
			return
		}
		switch r.URL.Path {
		case "/login":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
		case "/panel/api/xray/outbound-subs":
			_, _ = w.Write([]byte(`{"success":true,"obj":` + storedOutboundSub + `}`))
		case "/panel/api/xray/outbound-subs/4":
			if err := r.ParseForm(); err != nil {
				t.Fatalf("parsing form: %v", err)
			}
			got = r.PostForm
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	})

	_, err := h.updateOutboundSub(context.Background(), req(map[string]any{
		"id":     float64(4),
		"remark": "nl-pool-v2",
	}))
	if err != nil {
		t.Fatalf("updateOutboundSub returned error: %v", err)
	}

	if got.Get("remark") != "nl-pool-v2" {
		t.Errorf("remark = %q, want the new value", got.Get("remark"))
	}
	for field, want := range map[string]string{
		"url":            "https://example.com/sub",
		"tagPrefix":      "nl-",
		"enabled":        "true",
		"updateInterval": "900",
		"allowInsecure":  "true",
		"prepend":        "true",
		"allowPrivate":   "false",
	} {
		if got.Get(field) != want {
			t.Errorf("%s = %q, want %q — omitted fields must survive the update", field, got.Get(field), want)
		}
	}
}

func TestUpdateOutboundSub_UnknownID(t *testing.T) {
	h := newXrayHandler(t, func(w http.ResponseWriter, r *http.Request) {
		if writeCSRF(w, r) {
			return
		}
		switch r.URL.Path {
		case "/login":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true})
		case "/panel/api/xray/outbound-subs":
			_, _ = w.Write([]byte(`{"success":true,"obj":` + storedOutboundSub + `}`))
		default:
			t.Errorf("must not write when the subscription is unknown (path %s)", r.URL.Path)
		}
	})

	result, err := h.updateOutboundSub(context.Background(), req(map[string]any{"id": float64(99)}))
	if err != nil {
		t.Fatalf("updateOutboundSub returned error: %v", err)
	}
	if !result.IsError {
		t.Error("expected a tool error for an unknown subscription id")
	}
}
