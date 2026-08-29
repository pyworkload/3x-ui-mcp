package handler

import (
	"strings"
	"testing"
)

// The history routes reject any bucket the panel does not aggregate into, and
// the tools check it first so the caller gets the allowed list instead of a
// bare "invalid bucket" from the API. 120 and 300 are in the published
// openapi.json but not in the panel, which is what made this worth pinning.
func TestCheckBucket(t *testing.T) {
	for _, bucket := range historyBuckets {
		if err := checkBucket(bucket); err != nil {
			t.Errorf("checkBucket(%d) = %v, want nil", bucket, err)
		}
	}

	for _, bucket := range []int{0, 1, 45, 120, 300, 600, -60} {
		err := checkBucket(bucket)
		if err == nil {
			t.Errorf("checkBucket(%d) = nil, want an error", bucket)
			continue
		}
		if !strings.Contains(err.Error(), "10080") {
			t.Errorf("checkBucket(%d) error %q does not name the allowed buckets", bucket, err)
		}
	}
}
