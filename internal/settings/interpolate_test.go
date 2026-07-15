package settings

import (
	"strings"
	"testing"
)

func TestDeepMapWalksArraysAndMaps(t *testing.T) {
	vars := map[string]string{"stage": "dev"}
	got, err := deepMap(map[string]any{
		"list":   []any{"a-${stage}", 42, true},
		"nested": map[string]any{"inner": "${stage}"},
	}, vars)
	if err != nil {
		t.Fatal(err)
	}
	tree := got.(map[string]any)
	list := tree["list"].([]any)
	if list[0] != "a-dev" || list[1] != "42" || list[2] != "true" {
		t.Errorf("array scalars wrong: %v", list)
	}
	if tree["nested"].(map[string]any)["inner"] != "dev" {
		t.Errorf("nested map not interpolated: %v", tree)
	}
}

func TestDeepMapPropagatesErrors(t *testing.T) {
	vars := map[string]string{}
	if _, err := deepMap(map[string]any{"a": []any{"${nope}"}}, vars); err == nil || err.Error() != "nope is not defined" {
		t.Errorf("array error not propagated: %v", err)
	}
	if _, err := deepMap(map[string]any{"a": map[string]any{"b": "${nope}"}}, vars); err == nil {
		t.Errorf("map error not propagated")
	}
}

func TestStringifyUnsupportedType(t *testing.T) {
	if _, err := stringify(struct{}{}); err == nil || !strings.Contains(err.Error(), "unsupported value type") {
		t.Errorf("unsupported type must error: %v", err)
	}
}

func TestInterpolatedStacks(t *testing.T) {
	vars := map[string]string{"stage": "dev"}

	stacks, err := interpolatedStacks(map[string]any{"stacks": []any{"api-${stage}", 42}}, vars)
	if err != nil {
		t.Fatal(err)
	}
	if len(stacks) != 2 || stacks[0] != "api-dev" || stacks[1] != "42" {
		t.Errorf("stacks = %v", stacks)
	}

	// No stacks key (or a non-list) → nil, no error.
	if stacks, err := interpolatedStacks(map[string]any{}, vars); err != nil || stacks != nil {
		t.Errorf("absent stacks: %v, %v", stacks, err)
	}
	if stacks, err := interpolatedStacks(map[string]any{"stacks": "not-a-list"}, vars); err != nil || stacks != nil {
		t.Errorf("non-list stacks: %v, %v", stacks, err)
	}

	// Undefined variable in a stack name is a hard error.
	if _, err := interpolatedStacks(map[string]any{"stacks": []any{"${nope}"}}, vars); err == nil {
		t.Errorf("undefined var in stack name must error")
	}
}
