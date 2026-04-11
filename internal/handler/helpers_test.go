package handler

import (
	"encoding/json"
	"errors"
	"regexp"
	"testing"

	"github.com/pyworkload/3x-ui-mcp/internal/xui"
)

func TestToResult_SuccessResponse(t *testing.T) {
	resp := &xui.Response{
		Success: true,
		Msg:     "operation completed",
		Obj:     json.RawMessage(`{"id": 1}`),
	}

	result, err := toResult(resp, nil)
	if err != nil {
		t.Fatalf("toResult returned Go error: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	if result.IsError {
		t.Error("expected IsError to be false for success response")
	}
	// Check that the result content contains the formatted JSON
	if len(result.Content) == 0 {
		t.Fatal("result content is empty")
	}
}

func TestToResult_ErrorResponse(t *testing.T) {
	resp := &xui.Response{
		Success: false,
		Msg:     "inbound not found",
	}

	result, err := toResult(resp, nil)
	if err != nil {
		t.Fatalf("toResult returned Go error: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	if !result.IsError {
		t.Error("expected IsError to be true for API error response")
	}
	// Verify the error message contains the API message
	if len(result.Content) == 0 {
		t.Fatal("result content is empty")
	}
}

func TestToResult_GoError(t *testing.T) {
	goErr := errors.New("connection refused")

	result, err := toResult(nil, goErr)
	if err != nil {
		t.Fatalf("toResult should not return Go error, got: %v", err)
	}
	if result == nil {
		t.Fatal("result is nil")
	}
	if !result.IsError {
		t.Error("expected IsError to be true for Go error")
	}
}

func TestFormatResponse_WithJSONObj(t *testing.T) {
	resp := &xui.Response{
		Success: true,
		Msg:     "",
		Obj:     json.RawMessage(`{"name":"test","port":443}`),
	}

	got := formatResponse(resp)
	// Should be pretty-printed
	var parsed map[string]any
	if err := json.Unmarshal([]byte(got), &parsed); err != nil {
		t.Fatalf("formatResponse output is not valid JSON: %v\nGot: %s", err, got)
	}
	if parsed["name"] != "test" {
		t.Errorf("name = %v, want %q", parsed["name"], "test")
	}
	if parsed["port"] != float64(443) {
		t.Errorf("port = %v, want %v", parsed["port"], 443)
	}
	// Verify it's indented (pretty-printed)
	if got == `{"name":"test","port":443}` {
		t.Error("expected pretty-printed JSON, got compact JSON")
	}
}

func TestFormatResponse_NilObj(t *testing.T) {
	resp := &xui.Response{
		Success: true,
		Msg:     "",
		Obj:     nil,
	}

	got := formatResponse(resp)
	if got != `{"success": true}` {
		t.Errorf("formatResponse with nil Obj = %q, want %q", got, `{"success": true}`)
	}
}

func TestFormatResponse_NullObj(t *testing.T) {
	resp := &xui.Response{
		Success: true,
		Msg:     "",
		Obj:     json.RawMessage(`null`),
	}

	got := formatResponse(resp)
	if got != `{"success": true}` {
		t.Errorf("formatResponse with null Obj = %q, want %q", got, `{"success": true}`)
	}
}

func TestFormatResponse_NilObjWithMessage(t *testing.T) {
	resp := &xui.Response{
		Success: true,
		Msg:     "operation succeeded",
		Obj:     nil,
	}

	got := formatResponse(resp)
	if got != "operation succeeded" {
		t.Errorf("formatResponse with nil Obj and message = %q, want %q", got, "operation succeeded")
	}
}

func TestFormatResponse_EmptyMessage_NilObj(t *testing.T) {
	resp := &xui.Response{
		Success: true,
		Msg:     "",
		Obj:     nil,
	}

	got := formatResponse(resp)
	if got != `{"success": true}` {
		t.Errorf("formatResponse with empty msg and nil obj = %q, want %q", got, `{"success": true}`)
	}
}

func TestFormatResponse_ArrayObj(t *testing.T) {
	resp := &xui.Response{
		Success: true,
		Msg:     "",
		Obj:     json.RawMessage(`[1,2,3]`),
	}

	got := formatResponse(resp)
	var parsed []any
	if err := json.Unmarshal([]byte(got), &parsed); err != nil {
		t.Fatalf("formatResponse array output is not valid JSON: %v\nGot: %s", err, got)
	}
	if len(parsed) != 3 {
		t.Errorf("parsed array length = %d, want 3", len(parsed))
	}
}

func TestGenerateUUID_Format(t *testing.T) {
	uuid := generateUUID()
	// UUID v4 format: 8-4-4-4-12 hex chars
	uuidPattern := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

	if !uuidPattern.MatchString(uuid) {
		t.Errorf("generateUUID() = %q, does not match UUID v4 pattern", uuid)
	}
}

func TestGenerateUUID_Uniqueness(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		uuid := generateUUID()
		if seen[uuid] {
			t.Errorf("duplicate UUID generated: %s", uuid)
		}
		seen[uuid] = true
	}
}

func TestGenerateUUID_Version4Bits(t *testing.T) {
	// Generate several UUIDs and check version/variant bits
	for i := 0; i < 10; i++ {
		uuid := generateUUID()
		// Version nibble at position 14 (0-indexed) should be '4'
		if uuid[14] != '4' {
			t.Errorf("UUID %q: version nibble at [14] = %c, want '4'", uuid, uuid[14])
		}
		// Variant nibble at position 19 should be 8, 9, a, or b
		v := uuid[19]
		if v != '8' && v != '9' && v != 'a' && v != 'b' {
			t.Errorf("UUID %q: variant nibble at [19] = %c, want one of [89ab]", uuid, v)
		}
	}
}

func TestBuildClientSettings_ValidJSON(t *testing.T) {
	client := xui.ClientConfig{
		ID:         "test-uuid-1234",
		Email:      "user@example.com",
		Enable:     true,
		LimitIP:    2,
		TotalGB:    10737418240,
		ExpiryTime: 1700000000000,
		TgID:       0,
		SubID:      "sub1",
	}

	result, err := buildClientSettings(client)
	if err != nil {
		t.Fatalf("buildClientSettings returned error: %v", err)
	}

	// Parse the result to verify it's valid JSON
	var settings xui.InboundSettings
	if err := json.Unmarshal([]byte(result), &settings); err != nil {
		t.Fatalf("result is not valid JSON: %v\nGot: %s", err, result)
	}

	if len(settings.Clients) != 1 {
		t.Fatalf("expected 1 client, got %d", len(settings.Clients))
	}

	c := settings.Clients[0]
	if c.ID != "test-uuid-1234" {
		t.Errorf("client ID = %q, want %q", c.ID, "test-uuid-1234")
	}
	if c.Email != "user@example.com" {
		t.Errorf("client Email = %q, want %q", c.Email, "user@example.com")
	}
	if !c.Enable {
		t.Error("client Enable = false, want true")
	}
	if c.LimitIP != 2 {
		t.Errorf("client LimitIP = %d, want %d", c.LimitIP, 2)
	}
	if c.TotalGB != 10737418240 {
		t.Errorf("client TotalGB = %d, want %d", c.TotalGB, 10737418240)
	}
}

func TestBuildClientSettings_MinimalClient(t *testing.T) {
	client := xui.ClientConfig{
		Email:  "minimal@test.com",
		Enable: true,
	}

	result, err := buildClientSettings(client)
	if err != nil {
		t.Fatalf("buildClientSettings returned error: %v", err)
	}

	var settings xui.InboundSettings
	if err := json.Unmarshal([]byte(result), &settings); err != nil {
		t.Fatalf("result is not valid JSON: %v\nGot: %s", err, result)
	}

	if len(settings.Clients) != 1 {
		t.Fatalf("expected 1 client, got %d", len(settings.Clients))
	}
	if settings.Clients[0].Email != "minimal@test.com" {
		t.Errorf("client Email = %q, want %q", settings.Clients[0].Email, "minimal@test.com")
	}
}

func TestBuildClientSettings_StructureHasClientsArray(t *testing.T) {
	client := xui.ClientConfig{
		Email:  "check@struct.com",
		Enable: true,
	}

	result, err := buildClientSettings(client)
	if err != nil {
		t.Fatalf("buildClientSettings returned error: %v", err)
	}

	// Verify the JSON has "clients" key at top level
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(result), &raw); err != nil {
		t.Fatalf("result is not a JSON object: %v", err)
	}
	if _, ok := raw["clients"]; !ok {
		t.Error("JSON output missing 'clients' key")
	}
}
