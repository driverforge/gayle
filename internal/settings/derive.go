package settings

import "fmt"

// extract turns the interpolated tree into a typed Settings, deriving the
// declared parameter-name lists.
func extract(v any, stage string) (*Settings, error) {
	tree, ok := v.(map[string]any)
	if !ok {
		tree = map[string]any{}
	}

	s := &Settings{
		Service: strAt(tree, "service"),
		Provider: Provider{
			Name:  strAt(mapAt(tree, "provider"), "name"),
			Vault: strAt(mapAt(tree, "provider"), "vault"),
		},
	}

	if list, ok := tree["stacks"].([]any); ok {
		for _, item := range list {
			name, _ := item.(string)
			s.Stacks = append(s.Stacks, name)
		}
	}

	if rawConfig, ok := tree["config"].(map[string]any); ok {
		block := &ConfigBlock{
			Path:     strAt(rawConfig, "path"),
			Defaults: stringMap(rawConfig["defaults"]),
			Required: stringMap(rawConfig["required"]),
		}
		// The per-stage override block is a sibling of defaults/required, keyed
		// by the literal stage name. It only influences `run` (populateConfig);
		// its keys are deliberately NOT part of ConfigParameters (Node parity).
		switch stage {
		case "path", "defaults", "required":
			// A stage named after a fixed key can't have overrides.
		default:
			if overrides, ok := rawConfig[stage].(map[string]any); ok {
				block.StageOverrides = stringMap(overrides)
			}
		}
		s.Config = block
	}

	if rawSecret, ok := tree["secret"].(map[string]any); ok {
		s.Secret = &SecretBlock{
			KeyID:    strAt(rawSecret, "keyId"),
			Path:     strAt(rawSecret, "path"),
			Required: stringMap(rawSecret["required"]),
		}
	}

	if s.Config != nil {
		declared := make(map[string]string, len(s.Config.Defaults)+len(s.Config.Required))
		for k, v := range s.Config.Defaults {
			declared[k] = v
		}
		for k, v := range s.Config.Required {
			declared[k] = v
		}
		names, err := deriveParameters("config", s.Config.Path, declared)
		if err != nil {
			return nil, err
		}
		s.ConfigParameters = names
	}
	if s.Secret != nil {
		names, err := deriveParameters("secret", s.Secret.Path, s.Secret.Required)
		if err != nil {
			return nil, err
		}
		s.SecretParameters = names
	}
	return s, nil
}

func strAt(m map[string]any, key string) string {
	s, _ := m[key].(string)
	return s
}

// stringMap converts an interpolated subtree into a flat KEY→value map. After
// deepMap every leaf is a string; a nested map here means the yml shape is
// wrong, which is reported rather than dropped.
func stringMap(v any) map[string]string {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, item := range m {
		s, ok := item.(string)
		if !ok {
			s = fmt.Sprintf("%v", item)
		}
		out[k] = s
	}
	return out
}
