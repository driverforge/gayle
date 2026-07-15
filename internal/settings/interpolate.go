package settings

import (
	"fmt"
	"regexp"
	"strconv"
)

// The Node CLI interpolated with lodash _.template, whose ${...} evaluates
// arbitrary JavaScript. Every real gayle.yml only ever uses bare identifiers
// (${stage}, ${accountId}, ${UserPoolId}), so the port supports exactly that —
// and refuses anything else rather than silently mis-evaluating it.

var (
	placeholderRe = regexp.MustCompile(`\$\{[^}]*\}`)
	identifierRe  = regexp.MustCompile(`^[A-Za-z_$][A-Za-z0-9_$]*$`)
)

// interpolateString substitutes ${name} references from vars. An undefined
// name is a hard error (lodash threw a ReferenceError — same wording); a
// non-identifier expression is refused.
func interpolateString(s string, vars map[string]string) (string, error) {
	var firstErr error
	out := placeholderRe.ReplaceAllStringFunc(s, func(match string) string {
		name := match[2 : len(match)-1]
		if !identifierRe.MatchString(name) {
			if firstErr == nil {
				firstErr = fmt.Errorf("unsupported expression %s in gayle.yml: only ${variableName} references are supported", match)
			}
			return match
		}
		value, ok := vars[name]
		if !ok {
			if firstErr == nil {
				firstErr = fmt.Errorf("%s is not defined", name)
			}
			return match
		}
		return value
	})
	if firstErr != nil {
		return "", firstErr
	}
	return out, nil
}

// stringify coerces a YAML scalar the way lodash template did (JS string
// coercion): numbers and booleans become their JS string form, null becomes
// the empty string. DB_HOST: 3200 therefore lands remotely as "3200".
func stringify(v any) (string, error) {
	switch t := v.(type) {
	case string:
		return t, nil
	case int:
		return strconv.Itoa(t), nil
	case int64:
		return strconv.FormatInt(t, 10), nil
	case uint64:
		return strconv.FormatUint(t, 10), nil
	case float64:
		return strconv.FormatFloat(t, 'f', -1, 64), nil
	case bool:
		return strconv.FormatBool(t), nil
	case nil:
		return "", nil
	default:
		return "", fmt.Errorf("unsupported value type %T", v)
	}
}

// deepMap walks the parsed YAML tree, coercing every scalar leaf to a string
// and interpolating it — the Node CLI's deepMap(config, interpolate). Map keys
// are never interpolated, only values.
func deepMap(v any, vars map[string]string) (any, error) {
	switch t := v.(type) {
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, item := range t {
			mapped, err := deepMap(item, vars)
			if err != nil {
				return nil, err
			}
			out[k] = mapped
		}
		return out, nil
	case []any:
		out := make([]any, len(t))
		for i, item := range t {
			mapped, err := deepMap(item, vars)
			if err != nil {
				return nil, err
			}
			out[i] = mapped
		}
		return out, nil
	default:
		s, err := stringify(t)
		if err != nil {
			return nil, err
		}
		return interpolateString(s, vars)
	}
}
