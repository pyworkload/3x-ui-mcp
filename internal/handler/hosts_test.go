package handler

import (
	"testing"
)

// update_host_group is read-modify-write against a group the panel replaces
// wholesale: anything the caller did not mention has to come back unchanged,
// or a rename would quietly drop the group's hosts.
func TestHostGroupBody_OverlaysOnlySuppliedFields(t *testing.T) {
	stored := map[string]any{
		"groupId":    "grp-1",
		"remark":     "old name",
		"hosts":      []any{"cdn.example.com"},
		"port":       float64(8443),
		"security":   "tls",
		"inboundIds": []any{float64(3)},
		"isDisabled": false,
	}

	got, err := hostGroupBody(req(map[string]any{
		"group_id": "grp-1",
		"remark":   "new name",
	}), stored)
	if err != nil {
		t.Fatalf("hostGroupBody returned error: %v", err)
	}

	if got["remark"] != "new name" {
		t.Errorf("remark = %v, want the supplied value", got["remark"])
	}
	for key, want := range map[string]any{"security": "tls", "port": float64(8443)} {
		if got[key] != want {
			t.Errorf("%s = %v, want the stored %v", key, got[key], want)
		}
	}
	if hosts, ok := got["hosts"].([]any); !ok || len(hosts) != 1 {
		t.Errorf("hosts = %v, want the stored list preserved", got["hosts"])
	}
}

func TestHostGroupBody_SuppliedEmptyValuesStillApply(t *testing.T) {
	stored := map[string]any{"remark": "old", "sni": "example.com"}

	got, err := hostGroupBody(req(map[string]any{"sni": ""}), stored)
	if err != nil {
		t.Fatalf("hostGroupBody returned error: %v", err)
	}

	if got["sni"] != "" {
		t.Errorf("sni = %v, want the explicit empty value to clear it", got["sni"])
	}
	if got["remark"] != "old" {
		t.Errorf("remark = %v, want it untouched", got["remark"])
	}
}

// raw_json is the escape hatch for the fields the tool does not expose, so it
// has to win outright rather than merge into a half-built group.
func TestHostGroupBody_RawJSONReplacesEverything(t *testing.T) {
	got, err := hostGroupBody(req(map[string]any{
		"remark":   "ignored",
		"raw_json": `{"remark":"raw","inboundIds":[9],"muxParams":"{}"}`,
	}), map[string]any{"remark": "stored", "sni": "example.com"})
	if err != nil {
		t.Fatalf("hostGroupBody returned error: %v", err)
	}

	if got["remark"] != "raw" {
		t.Errorf("remark = %v, want the raw_json value", got["remark"])
	}
	if _, present := got["sni"]; present {
		t.Error("raw_json result carries a stored field, want only what raw_json declared")
	}
}

func TestHostGroupBody_RejectsBadRawJSON(t *testing.T) {
	_, err := hostGroupBody(req(map[string]any{"raw_json": "not json"}), map[string]any{})

	if err == nil {
		t.Fatal("hostGroupBody accepted invalid raw_json")
	}
}
