package handler

import (
	"strings"
	"testing"
)

// The generate_key tool advertises its types through an enum in the schema, so
// the dispatch table has to carry exactly those — a name in one and not the
// other is either an unreachable generator or a tool error the LLM can't avoid.
func TestKeyGenerators_MatchAdvertisedTypes(t *testing.T) {
	want := []string{"mldsa65", "mlkem768", "uuid", "vless_enc", "x25519"}

	got := keyGeneratorNames()

	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("generator names = %v, want %v", got, want)
	}
	for _, name := range want {
		if keyGenerators[name] == nil {
			t.Errorf("no generator registered for %q", name)
		}
	}
}
