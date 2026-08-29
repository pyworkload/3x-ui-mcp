package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/pyworkload/3x-ui-mcp/internal/config"
	"github.com/pyworkload/3x-ui-mcp/internal/xui"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func testServer(t *testing.T) (*server.MCPServer, *xui.Client) {
	t.Helper()
	cfg := &config.Config{Host: "http://panel.invalid", BasePath: "/", Username: "admin", Password: "admin"}
	return server.NewMCPServer("test", "0"), xui.NewClient(cfg, slog.Default())
}

func TestRegisterAll_UnknownToolsetNamesTheAvailableOnes(t *testing.T) {
	s, client := testServer(t)

	err := RegisterAll(s, client, []string{"clients", "nope"})
	if err == nil {
		t.Fatal("RegisterAll accepted an unknown toolset")
	}
	for _, want := range []string{"nope", "geodata", "metrics"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

// A narrowed selection is the whole point of XUI_TOOLSETS: the tools left out
// must not reach the client, since their schemas are what costs context.
func TestRegisterAll_SubsetRegistersOnlyRequestedGroups(t *testing.T) {
	s, client := testServer(t)

	if err := RegisterAll(s, client, []string{"metrics"}); err != nil {
		t.Fatalf("RegisterAll returned error: %v", err)
	}

	tools := s.ListTools()
	if _, present := tools["get_metrics_history"]; !present {
		t.Error("metrics toolset did not register get_metrics_history")
	}
	for _, unwanted := range []string{"list_inbounds", "add_client", "get_xray_template", "list_geodata_files"} {
		if _, present := tools[unwanted]; present {
			t.Errorf("%s was registered although only the metrics toolset was requested", unwanted)
		}
	}
}

// An empty selection means everything, and "everything" has to add up to the
// same set as registering each group by hand.
func TestRegisterAll_EmptyMeansEveryGroup(t *testing.T) {
	all, client := testServer(t)
	if err := RegisterAll(all, client, nil); err != nil {
		t.Fatalf("RegisterAll returned error: %v", err)
	}

	var sum int
	for _, name := range ToolsetNames() {
		one, c := testServer(t)
		if err := RegisterAll(one, c, []string{name}); err != nil {
			t.Fatalf("RegisterAll(%s) returned error: %v", name, err)
		}
		sum += len(one.ListTools())
	}

	if got := len(all.ListTools()); got != sum {
		t.Errorf("all toolsets = %d tools, sum of the groups = %d", got, sum)
	}
}

func TestSummarizeXrayConfig(t *testing.T) {
	raw := json.RawMessage(`{
		"inbounds": [{}, {}],
		"outbounds": [{}, {}, {}],
		"routing": {"rules": [{}, {}, {}, {}], "balancers": [{"tag": "LB-PROXY"}]},
		"dns": {"servers": ["1.1.1.1"]},
		"observatory": {"probeInterval": "10s"}
	}`)

	got := summarizeXrayConfig(raw)

	for _, want := range []string{"2 inbounds", "3 outbounds", "4 routing rules", "1 dns servers", "LB-PROXY", "Observatory: configured"} {
		if !strings.Contains(got, want) {
			t.Errorf("summary %q is missing %q", got, want)
		}
	}
	if strings.Contains(got, "Metrics") {
		t.Errorf("summary claims metrics on a config without them: %q", got)
	}
}

func TestSummarizeXrayConfig_UnparseableStillPointsAtTheResource(t *testing.T) {
	got := summarizeXrayConfig(json.RawMessage(`not json`))

	if !strings.Contains(got, "linked resource") {
		t.Errorf("summary %q does not tell the caller where to look", got)
	}
}

func TestSummarizeLinks(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want []string
	}{
		{
			name: "counts by scheme",
			raw:  `["vless://a@h:443#one","vless://b@h:443#two","trojan://c@h:443#three"]`,
			want: []string{"3 connection links", "vless: 2", "trojan: 1"},
		},
		{
			name: "empty panel",
			raw:  `[]`,
			want: []string{"No connection links"},
		},
		{
			name: "unparseable",
			raw:  `{"not":"an array"}`,
			want: []string{"linked resource"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := summarizeLinks(json.RawMessage(tt.raw))
			for _, want := range tt.want {
				if !strings.Contains(got, want) {
					t.Errorf("summary %q is missing %q", got, want)
				}
			}
			// Credentials must never ride along in the summary.
			if strings.Contains(got, "@h:443") {
				t.Errorf("summary leaks a connection URL: %q", got)
			}
		})
	}
}

func TestReadInboundResource_RejectsNonNumericID(t *testing.T) {
	_, client := testServer(t)
	h := &resourceHandler{client: client}

	_, err := h.readInbound(context.Background(), mcp.ReadResourceRequest{
		Params: mcp.ReadResourceParams{URI: "xui://inbound/abc"},
	})

	if err == nil {
		t.Fatal("readInbound accepted a non-numeric id")
	}
	if !strings.Contains(err.Error(), "numeric") {
		t.Errorf("error %q does not explain what is wrong", err)
	}
}

func TestStaticDocs_AreServedAsMarkdown(t *testing.T) {
	contents, err := staticResource(docsClientFieldsURI, "text/markdown", clientFieldsDoc)(context.Background(), mcp.ReadResourceRequest{})
	if err != nil {
		t.Fatalf("staticResource returned error: %v", err)
	}

	text, ok := contents[0].(mcp.TextResourceContents)
	if !ok {
		t.Fatalf("contents[0] is %T, want TextResourceContents", contents[0])
	}
	if text.MIMEType != "text/markdown" || text.URI != docsClientFieldsURI {
		t.Errorf("contents = %q %q, want markdown at %q", text.MIMEType, text.URI, docsClientFieldsURI)
	}
	// The docs exist to keep units out of every tool schema; if they stop
	// documenting the units, the schemas have to carry them again.
	for _, want := range []string{"milliseconds", "bytes"} {
		if !strings.Contains(text.Text, want) {
			t.Errorf("client field docs never mention %q", want)
		}
	}
}
