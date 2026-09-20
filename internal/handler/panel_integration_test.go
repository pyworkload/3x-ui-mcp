//go:build panel

// Read-modify-write contracts verified against a real 3x-ui panel rather than a
// mock. The tests that matter here are the ones a fixture cannot keep honest:
// the panel's update endpoints are full replaces, so "does an omitted field
// survive" is a question only the panel itself can answer, and its answer has
// already changed once under this code (v3.3.1 reshaped the inbound JSON, v3.8
// added columns).
//
// Run against a throwaway panel — these tests create and delete inbounds:
//
//	docker run -d --name 3xui_local -p 127.0.0.1:2053:2053 \
//	  -e XUI_INIT_WEB_BASE_PATH=/ ghcr.io/mhsanaei/3x-ui:v3.8.5
//	XUI_HOST=http://127.0.0.1:2053 go test -tags panel ./internal/handler/ -v
//
// XUI_USERNAME/XUI_PASSWORD default to the image's admin/admin.

package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/pyworkload/3x-ui-mcp/internal/config"
	"github.com/pyworkload/3x-ui-mcp/internal/xui"
)

func panelClient(t *testing.T) *xui.Client {
	t.Helper()
	host := os.Getenv("XUI_HOST")
	if host == "" {
		t.Skip("XUI_HOST is unset — start a throwaway panel first (see the file header)")
	}
	user, pass := os.Getenv("XUI_USERNAME"), os.Getenv("XUI_PASSWORD")
	if user == "" && pass == "" {
		user, pass = "admin", "admin"
	}
	basePath := os.Getenv("XUI_BASE_PATH")
	if basePath == "" {
		basePath = "/"
	}
	cfg := &config.Config{Host: host, BasePath: basePath, Username: user, Password: pass}
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelWarn}))
	return xui.NewClient(cfg, logger)
}

// call runs a panel API method and fails the test unless it succeeded at both
// levels: the HTTP call and the panel's own success flag.
func call(t *testing.T, what string, do func() (*xui.Response, error)) *xui.Response {
	t.Helper()
	resp, err := do()
	if err != nil {
		t.Fatalf("%s: %v", what, err)
	}
	if !resp.Success {
		t.Fatalf("%s: panel refused it: %s", what, resp.Msg)
	}
	return resp
}

// newInbound creates a VLESS/Reality inbound carrying every column this code
// has to round-trip, and returns its id. It is deleted when the test ends.
func newInbound(t *testing.T, c *xui.Client, port int) int {
	t.Helper()
	ctx := context.Background()

	keys := call(t, "generate_key", func() (*xui.Response, error) { return c.GetNewX25519Cert(ctx) })
	var pair struct {
		PrivateKey string `json:"privateKey"`
		Password   string `json:"password"`
	}
	if err := json.Unmarshal(keys.Obj, &pair); err != nil {
		t.Fatalf("parsing the generated key pair: %v", err)
	}
	privateKey := pair.PrivateKey
	if privateKey == "" {
		privateKey = pair.Password
	}

	settings := `{"clients":[],"decryption":"none"}`
	stream := fmt.Sprintf(`{"network":"tcp","security":"reality","realitySettings":{"show":false,"dest":"api.vk.com:443","target":"api.vk.com:443","xver":0,"serverNames":["api.vk.com"],"privateKey":%q,"shortIds":["83"],"settings":{"publicKey":"","fingerprint":"chrome","spiderX":"/"}},"tcpSettings":{"header":{"type":"none"}}}`, privateKey)

	resp := call(t, "create_inbound", func() (*xui.Response, error) {
		return c.CreateInbound(ctx, map[string]any{
			"remark":            "mcp-itest",
			"port":              port,
			"protocol":          "vless",
			"settings":          settings,
			"streamSettings":    stream,
			"sniffing":          `{"enabled":true,"destOverride":["http","tls"],"routeOnly":true}`,
			"listen":            "",
			"enable":            true,
			"expiryTime":        0,
			"total":             0,
			"subSortIndex":      7,
			"shareAddrStrategy": "listen",
			"trafficReset":      "monthly",
			"trafficResetDay":   14,
		})
	})

	var created struct {
		ID int `json:"id"`
	}
	if err := json.Unmarshal(resp.Obj, &created); err != nil || created.ID == 0 {
		t.Fatalf("no inbound id in the create response (%v): %s", err, resp.Obj)
	}
	t.Cleanup(func() {
		if _, err := c.DeleteInbound(context.Background(), created.ID); err != nil {
			t.Logf("cleaning up inbound %d: %v", created.ID, err)
		}
	})
	return created.ID
}

func currentInbound(t *testing.T, c *xui.Client, id int) map[string]any {
	t.Helper()
	resp := call(t, "get_inbound", func() (*xui.Response, error) { return c.GetInbound(context.Background(), id) })
	var m map[string]any
	if err := json.Unmarshal(resp.Obj, &m); err != nil {
		t.Fatalf("parsing inbound %d: %v", id, err)
	}
	return m
}

// A one-field update must leave every other column as it was. The panel replaces
// the whole row, so anything mergeInboundPatch fails to carry over is reset —
// which is how subSortIndex, trafficReset and disableFlow used to be lost.
func TestPanel_UpdateInboundKeepsEveryOtherColumn(t *testing.T) {
	c := panelClient(t)
	ctx := context.Background()
	id := newInbound(t, c, 24101)

	before := currentInbound(t, c, id)

	resp := call(t, "get_inbound", func() (*xui.Response, error) { return c.GetInbound(ctx, id) })
	merged, err := mergeInboundPatch(resp.Obj, map[string]any{"remark": "mcp-itest-renamed"})
	if err != nil {
		t.Fatalf("mergeInboundPatch: %v", err)
	}
	call(t, "update_inbound", func() (*xui.Response, error) { return c.UpdateInbound(ctx, id, merged) })

	after := currentInbound(t, c, id)

	if after["remark"] != "mcp-itest-renamed" {
		t.Errorf("remark = %v, want the rename to have applied", after["remark"])
	}
	// Counters and the client stats move on their own; everything else is ours.
	volatile := map[string]bool{"remark": true, "up": true, "down": true, "clientStats": true}
	for key, want := range before {
		if volatile[key] {
			continue
		}
		got, ok := after[key]
		if !ok {
			t.Errorf("%s disappeared from the inbound after the update", key)
			continue
		}
		if !sameJSON(want, got) {
			t.Errorf("%s = %s, want it unchanged at %s", key, mustJSON(got), mustJSON(want))
		}
	}
}

// The inbound's JSON columns arrive as nested objects from v3.3.1 on. Parsing
// them into string fields is what made update_inbound fail outright on v3.8.5,
// so assert on the shape the panel actually sends.
func TestPanel_InboundSettingsArriveAsObjects(t *testing.T) {
	c := panelClient(t)
	id := newInbound(t, c, 24102)

	current := currentInbound(t, c, id)
	for _, key := range []string{"settings", "streamSettings", "sniffing"} {
		if _, ok := current[key].(map[string]any); !ok {
			t.Errorf("%s = %T, want a nested object", key, current[key])
		}
	}
}

// Same contract on the client side, where the blast radius is worse: the panel
// writes reverse, adTag, group and limitHwid whether or not they were sent, so
// an omitted field is not merely stale but erased.
func TestPanel_UpdateClientKeepsFieldsWithoutAToolParameter(t *testing.T) {
	c := panelClient(t)
	ctx := context.Background()
	id := newInbound(t, c, 24103)

	const email = "mcp-itest-client"
	call(t, "add_client", func() (*xui.Response, error) {
		return c.AddClient(ctx, xui.ClientCreatePayload{
			Client:     xui.ClientConfig{Email: email, ID: generateUUID(), Enable: true},
			InboundIds: []int{id},
		})
	})
	t.Cleanup(func() {
		if _, err := c.DeleteClient(context.Background(), email, false); err != nil {
			t.Logf("cleaning up client %s: %v", email, err)
		}
	})

	// Set the fields update_client has no parameter for, the way the panel's own
	// editor would, then change one unrelated field through the tool's code path.
	seed, err := clientBaseFromRecord(call(t, "get_client", func() (*xui.Response, error) { return c.GetClient(ctx, email) }).Obj)
	if err != nil {
		t.Fatalf("clientBaseFromRecord: %v", err)
	}
	seed["adTag"] = "0123456789abcdef0123456789abcdef"
	seed["limitHwid"] = 3
	seed["resetDay"] = 7
	seed["resetMax"] = 4
	seed["trafficReset"] = "monthly"
	seed["trafficResetDay"] = 14
	seed["reverse"] = map[string]any{"tag": "mcp-itest-reverse"}
	call(t, "update_client (seed)", func() (*xui.Response, error) { return c.UpdateClient(ctx, email, seed, nil) })

	before := call(t, "get_client", func() (*xui.Response, error) { return c.GetClient(ctx, email) })
	body, err := clientBaseFromRecord(before.Obj)
	if err != nil {
		t.Fatalf("clientBaseFromRecord: %v", err)
	}
	body["comment"] = "renewed"
	call(t, "update_client", func() (*xui.Response, error) { return c.UpdateClient(ctx, email, body, nil) })

	wantUUID := recordOf(t, before)["uuid"]
	got := clientRecord(t, c, email)
	if got["comment"] != "renewed" {
		t.Errorf("comment = %v, want the edit to have applied", got["comment"])
	}
	for key, want := range map[string]any{
		"adTag":           "0123456789abcdef0123456789abcdef",
		"limitHwid":       float64(3),
		"resetDay":        float64(7),
		"resetMax":        float64(4),
		"trafficReset":    "monthly",
		"trafficResetDay": float64(14),
	} {
		if got[key] != want {
			t.Errorf("%s = %v, want %v — the panel cleared it", key, got[key], want)
		}
	}
	reverse, ok := got["reverse"].(map[string]any)
	if !ok || reverse["tag"] != "mcp-itest-reverse" {
		t.Errorf("reverse = %v, want {tag: mcp-itest-reverse}", got["reverse"])
	}
	if got["uuid"] != wantUUID {
		t.Errorf("uuid = %v, want it preserved as %v", got["uuid"], wantUUID)
	}
}

// The premise behind every read-modify-write in this package: the panel does
// not merge. Sending a body without adTag and reverse clears both columns, so
// a tool that posted only its own parameters would erase whatever the panel's
// own editor had set. Kept as a test so the day the panel starts merging is
// noticed here rather than guessed at.
func TestPanel_UpdateClientClearsOmittedFields(t *testing.T) {
	c := panelClient(t)
	ctx := context.Background()
	id := newInbound(t, c, 24104)

	const email = "mcp-itest-nomerge"
	call(t, "add_client", func() (*xui.Response, error) {
		return c.AddClient(ctx, xui.ClientCreatePayload{
			Client:     xui.ClientConfig{Email: email, ID: generateUUID(), Enable: true},
			InboundIds: []int{id},
		})
	})
	t.Cleanup(func() {
		if _, err := c.DeleteClient(context.Background(), email, false); err != nil {
			t.Logf("cleaning up client %s: %v", email, err)
		}
	})

	seed, err := clientBaseFromRecord(call(t, "get_client", func() (*xui.Response, error) { return c.GetClient(ctx, email) }).Obj)
	if err != nil {
		t.Fatalf("clientBaseFromRecord: %v", err)
	}
	seed["adTag"] = "0123456789abcdef0123456789abcdef"
	seed["reverse"] = map[string]any{"tag": "mcp-itest-reverse"}
	seed["limitHwid"] = 3
	call(t, "update_client (seed)", func() (*xui.Response, error) { return c.UpdateClient(ctx, email, seed, nil) })

	if got := clientRecord(t, c, email); got["adTag"] != seed["adTag"] {
		t.Fatalf("the seed did not take: adTag = %v", got["adTag"])
	}

	// Now the naive body a tool would send if it only posted its own parameters.
	call(t, "update_client (naive)", func() (*xui.Response, error) {
		return c.UpdateClient(ctx, email, map[string]any{
			"email":   email,
			"comment": "renewed",
			"enable":  true,
		}, nil)
	})

	got := clientRecord(t, c, email)
	if got["adTag"] != "" {
		t.Errorf("adTag = %v, want it cleared — the panel is expected NOT to merge", got["adTag"])
	}
	if got["reverse"] != nil {
		t.Errorf("reverse = %v, want it cleared — the panel is expected NOT to merge", got["reverse"])
	}
	if got["limitHwid"] != float64(0) {
		t.Errorf("limitHwid = %v, want it reset — the panel is expected NOT to merge", got["limitHwid"])
	}
}

// The other client tests pin what survives an update. This one pins the other
// half: that every parameter add_client and update_client expose actually
// lands in the column it names. The panel validates several of them (adTag is
// 32 hex characters, trafficReset is an enum), so a wrong spelling here is a
// refusal rather than a silent no-op — which is the point of running it live.
func TestPanel_ClientFieldParametersReachTheirColumns(t *testing.T) {
	c := panelClient(t)
	ctx := context.Background()
	id := newInbound(t, c, 24105)
	h := &clientHandler{client: c}

	const email = "mcp-itest-fields"
	t.Cleanup(func() {
		if _, err := c.DeleteClient(context.Background(), email, false); err != nil {
			t.Logf("cleaning up client %s: %v", email, err)
		}
	})

	created, err := h.add(ctx, req(map[string]any{
		"inbound_ids":       []any{float64(id)},
		"email":             email,
		"limit_hwid":        float64(3),
		"reset_day":         float64(7),
		"reset_max":         float64(4),
		"traffic_reset":     "monthly",
		"traffic_reset_day": float64(14),
		"reverse_tag":       "mcp-itest-reverse",
		"ad_tag":            "0123456789abcdef0123456789abcdef",
		"secret":            "0123456789abcdef0123456789abcdef",
		"private_key":       "mcp-itest-private",
		"public_key":        "mcp-itest-public",
		"pre_shared_key":    "mcp-itest-psk",
		"allowed_ips":       []any{"10.0.0.2/32", "fd00::2/128"},
		"keep_alive":        float64(25),
		"forwarded_ports":   "80,443,8000-8100",
	}))
	if err != nil {
		t.Fatalf("add_client: %v", err)
	}
	if created.IsError {
		t.Fatalf("add_client: the panel refused it: %+v", created.Content)
	}

	got := clientRecord(t, c, email)
	for key, want := range map[string]any{
		"limitHwid":       float64(3),
		"resetDay":        float64(7),
		"resetMax":        float64(4),
		"trafficReset":    "monthly",
		"trafficResetDay": float64(14),
		"adTag":           "0123456789abcdef0123456789abcdef",
		"secret":          "0123456789abcdef0123456789abcdef",
		"privateKey":      "mcp-itest-private",
		"publicKey":       "mcp-itest-public",
		"preSharedKey":    "mcp-itest-psk",
		"keepAlive":       float64(25),
		"forwardedPorts":  "80,443,8000-8100",
		"allowedIPs":      "10.0.0.2/32,fd00::2/128",
	} {
		if got[key] != want {
			t.Errorf("after add_client, %s = %v, want %v", key, got[key], want)
		}
	}
	if reverse, ok := got["reverse"].(map[string]any); !ok || reverse["tag"] != "mcp-itest-reverse" {
		t.Errorf("after add_client, reverse = %v, want {tag: mcp-itest-reverse}", got["reverse"])
	}

	// Now edit a few of them through update_client and leave the rest alone:
	// the overlay has to change exactly what it was given.
	updated, err := h.update(ctx, req(map[string]any{
		"email":         email,
		"limit_hwid":    float64(9),
		"traffic_reset": "weekly",
		"keep_alive":    float64(0),
		"reverse_tag":   "",
	}))
	if err != nil {
		t.Fatalf("update_client: %v", err)
	}
	if updated.IsError {
		t.Fatalf("update_client: the panel refused it: %+v", updated.Content)
	}

	got = clientRecord(t, c, email)
	for key, want := range map[string]any{
		"limitHwid":    float64(9),
		"trafficReset": "weekly",
		"keepAlive":    float64(0),
		// Untouched by the update, so still what add_client set.
		"resetDay":       float64(7),
		"adTag":          "0123456789abcdef0123456789abcdef",
		"privateKey":     "mcp-itest-private",
		"forwardedPorts": "80,443,8000-8100",
		"allowedIPs":     "10.0.0.2/32,fd00::2/128",
	} {
		if got[key] != want {
			t.Errorf("after update_client, %s = %v, want %v", key, got[key], want)
		}
	}
	if got["reverse"] != nil {
		t.Errorf("after update_client, reverse = %v, want an empty reverse_tag to have cleared it", got["reverse"])
	}
}

func clientRecord(t *testing.T, c *xui.Client, email string) map[string]any {
	t.Helper()
	return recordOf(t, call(t, "get_client", func() (*xui.Response, error) { return c.GetClient(context.Background(), email) }))
}

// recordOf pulls the "client" object out of a clients/get response.
func recordOf(t *testing.T, resp *xui.Response) map[string]any {
	t.Helper()
	var wrap struct {
		Client map[string]any `json:"client"`
	}
	if err := json.Unmarshal(resp.Obj, &wrap); err != nil {
		t.Fatalf("parsing the client record: %v", err)
	}
	return wrap.Client
}

func mustJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}

func sameJSON(a, b any) bool { return mustJSON(a) == mustJSON(b) }
