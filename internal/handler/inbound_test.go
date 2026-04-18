package handler

import (
	"encoding/json"
	"strings"
	"testing"
)

// sampleInbound is a realistic GetInbound response body, before any patch.
const sampleInbound = `{
	"id": 5,
	"up": 1234,
	"down": 5678,
	"total": 0,
	"remark": "vless-xhttp-metrics",
	"enable": true,
	"expiryTime": 0,
	"listen": "",
	"port": 8200,
	"protocol": "vless",
	"settings": "{\"clients\":[{\"id\":\"uuid-1\",\"email\":\"user1\",\"enable\":true}],\"decryption\":\"none\"}",
	"streamSettings": "{\"network\":\"xhttp\",\"security\":\"none\",\"xhttpSettings\":{\"mode\":\"auto\",\"path\":\"/metrics\"}}",
	"tag": "inbound-8200",
	"sniffing": "{\"enabled\":true}",
	"clientStats": [
		{"id": 1, "inboundId": 5, "email": "user1", "up": 10, "down": 20, "total": 0, "enable": true}
	]
}`

func TestNormalizeInboundPatchKeys_RenamesSnakeCase(t *testing.T) {
	in := map[string]any{
		"stream_settings": "x",
		"expiry_time":     int64(123),
		"remark":          "keep",
	}
	out := normalizeInboundPatchKeys(in)

	if _, ok := out["stream_settings"]; ok {
		t.Error("expected stream_settings to be renamed away")
	}
	if _, ok := out["expiry_time"]; ok {
		t.Error("expected expiry_time to be renamed away")
	}
	if out["streamSettings"] != "x" {
		t.Errorf("streamSettings = %v, want %q", out["streamSettings"], "x")
	}
	if out["expiryTime"] != int64(123) {
		t.Errorf("expiryTime = %v, want %d", out["expiryTime"], 123)
	}
	if out["remark"] != "keep" {
		t.Errorf("remark = %v, want %q", out["remark"], "keep")
	}
}

func TestNormalizeInboundPatchKeys_PassesCamelCaseThrough(t *testing.T) {
	in := map[string]any{
		"streamSettings": "ss",
		"expiryTime":     int64(999),
	}
	out := normalizeInboundPatchKeys(in)
	if out["streamSettings"] != "ss" {
		t.Errorf("streamSettings = %v, want %q", out["streamSettings"], "ss")
	}
	if out["expiryTime"] != int64(999) {
		t.Errorf("expiryTime = %v, want %d", out["expiryTime"], 999)
	}
}

func TestNormalizeInboundPatchKeys_NilInput(t *testing.T) {
	out := normalizeInboundPatchKeys(nil)
	if out == nil {
		t.Fatal("expected non-nil map for nil input")
	}
	if len(out) != 0 {
		t.Errorf("expected empty map, got %d keys", len(out))
	}
}

func TestInboundBaseFromResponse_StripsClientStats(t *testing.T) {
	base, err := inboundBaseFromResponse(json.RawMessage(sampleInbound))
	if err != nil {
		t.Fatalf("inboundBaseFromResponse: %v", err)
	}
	if _, ok := base["clientStats"]; ok {
		t.Error("clientStats should be stripped from the update body")
	}
	if base["port"] != float64(8200) {
		t.Errorf("port = %v, want 8200", base["port"])
	}
	if base["protocol"] != "vless" {
		t.Errorf("protocol = %v, want %q", base["protocol"], "vless")
	}
	if base["remark"] != "vless-xhttp-metrics" {
		t.Errorf("remark = %v, want %q", base["remark"], "vless-xhttp-metrics")
	}
}

func TestInboundBaseFromResponse_EmptyObject(t *testing.T) {
	if _, err := inboundBaseFromResponse(nil); err == nil {
		t.Error("expected error for nil obj")
	}
	if _, err := inboundBaseFromResponse(json.RawMessage(`null`)); err == nil {
		t.Error("expected error for null obj")
	}
}

func TestMergeInboundPatch_OnlyStreamSettingsSnakeCase(t *testing.T) {
	patch := map[string]any{
		"stream_settings": `{"network":"xhttp","xhttpSettings":{"mode":"packet-up"}}`,
	}
	merged, err := mergeInboundPatch(json.RawMessage(sampleInbound), patch)
	if err != nil {
		t.Fatalf("mergeInboundPatch: %v", err)
	}

	// The one field the user wanted to change:
	if !strings.Contains(merged["streamSettings"].(string), "packet-up") {
		t.Errorf("streamSettings not updated: %v", merged["streamSettings"])
	}

	// All the other fields must survive untouched.
	if merged["port"] != float64(8200) {
		t.Errorf("port = %v, want 8200 (would be zeroed by the old bug)", merged["port"])
	}
	if merged["protocol"] != "vless" {
		t.Errorf("protocol = %v, want %q", merged["protocol"], "vless")
	}
	if merged["listen"] != "" {
		t.Errorf("listen = %v, want empty string", merged["listen"])
	}
	if merged["tag"] != "inbound-8200" {
		t.Errorf("tag = %v, want %q (would be reset to inbound-0 by the old bug)", merged["tag"], "inbound-8200")
	}
	if merged["remark"] != "vless-xhttp-metrics" {
		t.Errorf("remark = %v, want %q", merged["remark"], "vless-xhttp-metrics")
	}
	if merged["settings"] == "" {
		t.Error("settings should survive merge (not be blanked)")
	}

	// The snake_case key must NOT appear in the outbound body.
	if _, ok := merged["stream_settings"]; ok {
		t.Error("stream_settings (snake_case) leaked into the body — 3x-ui would ignore it and blank streamSettings")
	}
}

func TestMergeInboundPatch_OnlyStreamSettingsCamelCase(t *testing.T) {
	patch := map[string]any{
		"streamSettings": `{"network":"xhttp","xhttpSettings":{"mode":"packet-up"}}`,
	}
	merged, err := mergeInboundPatch(json.RawMessage(sampleInbound), patch)
	if err != nil {
		t.Fatalf("mergeInboundPatch: %v", err)
	}
	if !strings.Contains(merged["streamSettings"].(string), "packet-up") {
		t.Errorf("streamSettings not updated: %v", merged["streamSettings"])
	}
	if merged["port"] != float64(8200) {
		t.Errorf("port = %v, want 8200", merged["port"])
	}
}

func TestMergeInboundPatch_ExpiryTimeSnakeCase(t *testing.T) {
	patch := map[string]any{
		"expiry_time": float64(1700000000000),
	}
	merged, err := mergeInboundPatch(json.RawMessage(sampleInbound), patch)
	if err != nil {
		t.Fatalf("mergeInboundPatch: %v", err)
	}
	if merged["expiryTime"] != float64(1700000000000) {
		t.Errorf("expiryTime = %v, want 1700000000000", merged["expiryTime"])
	}
	if _, ok := merged["expiry_time"]; ok {
		t.Error("expiry_time (snake_case) leaked into the body")
	}
}

func TestMergeInboundPatch_RemarkOnly(t *testing.T) {
	patch := map[string]any{"remark": "new name"}
	merged, err := mergeInboundPatch(json.RawMessage(sampleInbound), patch)
	if err != nil {
		t.Fatalf("mergeInboundPatch: %v", err)
	}
	if merged["remark"] != "new name" {
		t.Errorf("remark = %v, want %q", merged["remark"], "new name")
	}
	if merged["protocol"] != "vless" {
		t.Errorf("protocol should not change: %v", merged["protocol"])
	}
}

func TestMergeInboundPatch_EmptyPatch(t *testing.T) {
	merged, err := mergeInboundPatch(json.RawMessage(sampleInbound), nil)
	if err != nil {
		t.Fatalf("mergeInboundPatch: %v", err)
	}
	// Empty patch should leave the inbound identical to current state.
	if merged["port"] != float64(8200) {
		t.Errorf("port = %v, want 8200", merged["port"])
	}
	if merged["protocol"] != "vless" {
		t.Errorf("protocol = %v, want %q", merged["protocol"], "vless")
	}
}

func TestMergeInboundPatch_RejectsPortZero(t *testing.T) {
	patch := map[string]any{"port": 0}
	_, err := mergeInboundPatch(json.RawMessage(sampleInbound), patch)
	if err == nil {
		t.Fatal("expected error when merge yields port=0")
	}
	if !strings.Contains(err.Error(), "port") {
		t.Errorf("error should mention port, got: %v", err)
	}
}

func TestMergeInboundPatch_RejectsEmptyProtocol(t *testing.T) {
	patch := map[string]any{"protocol": ""}
	_, err := mergeInboundPatch(json.RawMessage(sampleInbound), patch)
	if err == nil {
		t.Fatal("expected error when merge yields empty protocol")
	}
	if !strings.Contains(err.Error(), "protocol") {
		t.Errorf("error should mention protocol, got: %v", err)
	}
}

func TestMergeInboundPatch_RejectsBrokenBase(t *testing.T) {
	// If the GET response somehow returned a port=0 inbound, the validator
	// should block us from echoing that back and confirming the damage.
	broken := `{"id":5,"port":0,"protocol":"","listen":"","enable":false,"settings":"","streamSettings":"","tag":"inbound-0","sniffing":""}`
	_, err := mergeInboundPatch(json.RawMessage(broken), map[string]any{"remark": "rename"})
	if err == nil {
		t.Fatal("expected error when base inbound already has port=0")
	}
}

func TestValidateMergedInbound_AcceptsHealthy(t *testing.T) {
	m := map[string]any{"port": float64(8200), "protocol": "vless"}
	if err := validateMergedInbound(m); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateMergedInbound_RejectsMissingPort(t *testing.T) {
	m := map[string]any{"protocol": "vless"}
	if err := validateMergedInbound(m); err == nil {
		t.Error("expected error when port is missing")
	}
}

func TestNumberAsInt_HandlesManyTypes(t *testing.T) {
	cases := []struct {
		name string
		in   any
		want int
		ok   bool
	}{
		{"float64", float64(42), 42, true},
		{"int", 42, 42, true},
		{"int64", int64(42), 42, true},
		{"int32", int32(42), 42, true},
		{"json.Number", json.Number("42"), 42, true},
		{"json.Number invalid", json.Number("abc"), 0, false},
		{"string", "42", 0, false},
		{"nil", nil, 0, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := numberAsInt(tc.in)
			if ok != tc.ok {
				t.Errorf("ok = %v, want %v", ok, tc.ok)
			}
			if got != tc.want {
				t.Errorf("value = %d, want %d", got, tc.want)
			}
		})
	}
}
