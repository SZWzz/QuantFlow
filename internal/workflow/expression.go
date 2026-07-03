package workflow

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// exprPattern matches {{ $node_id.port }} or {{ $node_id.port.field }}
// Also supports simple arithmetic: {{ $a.x + $b.y }} or {{ $node_id.port + 0.01 }}
var exprPattern = regexp.MustCompile(`\{\{\s*(\$[^}]+)\}\}`)

// ResolveExpressions walks all node params and replaces {{...}} references
// with actual values from upstream outputs. Supports:
//
//	{{ $node_id.port }}         → value of that port output
//	{{ $node_id.port.field }}   → sub-field of that port (for map outputs)
func ResolveExpressions(params map[string]any, upstreamOutputs map[string]map[string]any, nodeID string) (map[string]any, error) {
	if len(params) == 0 {
		return params, nil
	}

	resolved := make(map[string]any, len(params))
	for key, val := range params {
		strVal, ok := val.(string)
		if !ok {
			resolved[key] = val
			continue
		}

		matches := exprPattern.FindAllStringSubmatch(strVal, -1)
		if len(matches) == 0 {
			resolved[key] = val
			continue
		}

		// Single expression {{ ... }} → replace the entire value
		if len(matches) == 1 && strings.TrimSpace(strVal) == matches[0][0] {
			replaced, err := evaluateExpr(matches[0][1], upstreamOutputs)
			if err != nil {
				return nil, fmt.Errorf("resolve param %q on node %q: %w", key, nodeID, err)
			}
			resolved[key] = replaced
		} else {
			// Multiple expressions → string replacement
			result := strVal
			for _, m := range matches {
				replaced, err := evaluateExpr(m[1], upstreamOutputs)
				if err != nil {
					return nil, fmt.Errorf("resolve param %q on node %q: %w", key, nodeID, err)
				}
				result = strings.Replace(result, m[0], fmt.Sprint(replaced), 1)
			}
			resolved[key] = result
		}
	}
	return resolved, nil
}

// evaluateExpr resolves a single expression like "$node_id.port" or "$node_id.port.field"
func evaluateExpr(expr string, upstreamOutputs map[string]map[string]any) (any, error) {
	expr = strings.TrimSpace(expr)
	if !strings.HasPrefix(expr, "$") {
		return expr, nil
	}

	// Strip leading $
	ref := expr[1:]

	// Check for arithmetic: split on + or -
	// Only support simple: $a.port + number
	for _, op := range []string{" + ", " - "} {
		if idx := strings.Index(ref, op); idx > 0 {
			left, err := resolveRef(strings.TrimSpace(ref[:idx]), upstreamOutputs)
			if err != nil {
				return nil, err
			}
			right, err := strconv.ParseFloat(strings.TrimSpace(ref[idx+len(op):]), 64)
			if err != nil {
				return nil, fmt.Errorf("right side of %q is not a number: %s", op, ref[idx+len(op):])
			}
			leftNum, err := toFloat64(left)
			if err != nil {
				return nil, fmt.Errorf("left side of %q is not a number: %v", op, left)
			}
			if strings.Contains(op, "-") {
				return leftNum - right, nil
			}
			return leftNum + right, nil
		}
	}

	return resolveRef(ref, upstreamOutputs)
}

// resolveRef resolves "$node_id.port" or "$node_id.port.field" from upstream outputs.
func resolveRef(ref string, upstreamOutputs map[string]map[string]any) (any, error) {
	// Parse: node_id.port or node_id.port.field
	parts := strings.SplitN(ref, ".", 3)
	if len(parts) < 2 {
		return nil, fmt.Errorf("invalid reference %q: expected $node_id.port or $node_id.port.field", ref)
	}

	nodeID := parts[0]
	portName := parts[1]
	fieldPath := ""
	if len(parts) > 2 {
		fieldPath = parts[2]
	}

	nodeOutputs, ok := upstreamOutputs[nodeID]
	if !ok {
		return nil, fmt.Errorf("node %q not found in upstream outputs", nodeID)
	}

	val, ok := nodeOutputs[portName]
	if !ok {
		return nil, fmt.Errorf("port %q not found on node %q (available: %v)", portName, nodeID, portKeys(nodeOutputs))
	}

	// Navigate sub-fields
	if fieldPath != "" {
		return navigateField(val, fieldPath)
	}
	return val, nil
}

// navigateField extracts a nested field from a value using dot notation (e.g., "data.close")
func navigateField(val any, path string) (any, error) {
	parts := strings.Split(path, ".")
	current := val
	for _, part := range parts {
		m, ok := current.(map[string]any)
		if !ok {
			// Try array index
			if idx, err := strconv.Atoi(part); err == nil {
				arr, ok2 := current.([]any)
				if ok2 && idx >= 0 && idx < len(arr) {
					current = arr[idx]
					continue
				}
				arr2, ok2 := current.([]float64)
				if ok2 && idx >= 0 && idx < len(arr2) {
					current = arr2[idx]
					continue
				}
			}
			return nil, fmt.Errorf("cannot access field %q on %T value", part, current)
		}
		v, ok := m[part]
		if !ok {
			return nil, fmt.Errorf("field %q not found (available: %v)", part, mapKeys(m))
		}
		current = v
	}
	return current, nil
}

func portKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func toFloat64(v any) (float64, error) {
	switch val := v.(type) {
	case float64:
		return val, nil
	case int:
		return float64(val), nil
	case []float64:
		if len(val) > 0 {
			return val[0], nil
		}
		return 0, fmt.Errorf("empty array")
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", v)
	}
}
