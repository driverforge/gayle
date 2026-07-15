package paramstore

import (
	"fmt"
	"strings"
)

// KeyError is one parameter's failure within a batch operation.
type KeyError struct {
	Key string
	Err error
}

// KeyErrors aggregates per-key failures from a write or delete batch. The
// operation attempts every key and returns everything that failed, so a
// partial failure reports the full damage in one run — and exits non-zero.
type KeyErrors []KeyError

func (ke KeyErrors) Error() string {
	if len(ke) == 1 {
		return fmt.Sprintf("1 parameter failed: %s: %v", ke[0].Key, ke[0].Err)
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%d parameters failed:", len(ke))
	for _, e := range ke {
		fmt.Fprintf(&b, "\n  %s: %v", e.Key, e.Err)
	}
	return b.String()
}

// OrNil returns ke as an error, or nil when no key failed — so callers can
// `return results, ke.OrNil()` without a typed-nil-in-interface bug.
func (ke KeyErrors) OrNil() error {
	if len(ke) == 0 {
		return nil
	}
	return ke
}
