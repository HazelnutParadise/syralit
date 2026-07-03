package insyradsl

import (
	"fmt"
	"strings"
)

// propString reads a string prop, returning "" when absent or not a string.
func propString(props map[string]any, key string) string {
	if props == nil {
		return ""
	}
	s, _ := props[key].(string)
	return strings.TrimSpace(s)
}

// propInt reads an integer prop, tolerating the float64 that JSON decoding
// produces. Returns 0 when absent or non-numeric.
func propInt(props map[string]any, key string) int {
	if props == nil {
		return 0
	}
	switch v := props[key].(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case float32:
		return int(v)
	}
	return 0
}

// propStringSlice reads a list-of-strings prop. It accepts a comma-separated
// string, a []string, or a []any (each element stringified) — covering both
// hand-written Go specs and JSON-decoded agent specs.
func propStringSlice(props map[string]any, key string) []string {
	if props == nil {
		return nil
	}
	switch v := props[key].(type) {
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return nil
		}
		parts := strings.Split(v, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
		return out
	case []string:
		return v
	case []any:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s := strings.TrimSpace(fmt.Sprint(item)); s != "" {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}
