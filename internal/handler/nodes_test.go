package handler

import (
	"strings"
	"testing"
)

// update_node is read-modify-write against a panel that replaces the row and
// validates name/address/port on every request, so the stored values have to
// come back even when the caller only flips one flag.
func TestNodeBody_OverlaysOnlySuppliedFields(t *testing.T) {
	stored := map[string]any{
		"name":            "de-fra-1",
		"address":         "node1.example.com",
		"port":            float64(2053),
		"scheme":          "https",
		"enable":          true,
		"tlsVerifyMode":   "pin",
		"inboundSyncMode": "all",
	}

	got, err := nodeBody(req(map[string]any{"id": float64(1), "enable": false}), stored)
	if err != nil {
		t.Fatalf("nodeBody returned error: %v", err)
	}

	if got["enable"] != false {
		t.Errorf("enable = %v, want the supplied false", got["enable"])
	}
	for key, want := range map[string]any{"name": "de-fra-1", "address": "node1.example.com", "tlsVerifyMode": "pin"} {
		if got[key] != want {
			t.Errorf("%s = %v, want the stored %v", key, got[key], want)
		}
	}
}

// The API token is write-only: the panel never returns it, so it must only ever
// be sent when the caller supplied one — otherwise an update would blank it.
func TestNodeBody_OmitsAPITokenUnlessSupplied(t *testing.T) {
	got, err := nodeBody(req(map[string]any{"remark": "moved"}), map[string]any{"name": "de-fra-1"})
	if err != nil {
		t.Fatalf("nodeBody returned error: %v", err)
	}

	if _, present := got["apiToken"]; present {
		t.Errorf("body carries apiToken = %v, want it omitted so the panel keeps the stored one", got["apiToken"])
	}
}

func TestNodeBody_RejectsTokenAndClearTogether(t *testing.T) {
	_, err := nodeBody(req(map[string]any{
		"api_token":       "abc",
		"clear_api_token": true,
	}), map[string]any{})

	if err == nil {
		t.Fatal("nodeBody accepted api_token together with clear_api_token")
	}
	if !strings.Contains(err.Error(), "mutually exclusive") {
		t.Errorf("error %q does not explain the conflict", err)
	}
}

// A node view carries heartbeat and counter fields the mutation validator would
// reject, so only the connection details may be echoed back.
func TestStoredNodeFields_DropsHeartbeatData(t *testing.T) {
	view := map[string]any{
		"name":        "de-fra-1",
		"address":     "node1.example.com",
		"port":        float64(2053),
		"cpuPct":      12.5,
		"activeCount": float64(20),
		"createdAt":   float64(1700000000),
		"hasApiToken": true,
	}

	got := storedNodeFields(view)

	for _, want := range []string{"name", "address", "port"} {
		if _, ok := got[want]; !ok {
			t.Errorf("connection field %q was dropped", want)
		}
	}
	for _, unwanted := range []string{"cpuPct", "activeCount", "createdAt", "hasApiToken"} {
		if _, present := got[unwanted]; present {
			t.Errorf("heartbeat field %q leaked into the mutation body", unwanted)
		}
	}
}
