package nodes

import (
	"fmt"
	"reflect"
)

// extractFloatSlice is a convenience alias for extractFloat64Slice.
func extractFloatSlice(val any) []float64 {
	return extractFloat64Slice(val)
}

// extractFloat64Slice converts any slice-like value to []float64.
// Supports: []float64, []any, and any []struct with a float64 Close field (e.g., OHLCVBar).
func extractFloat64Slice(raw any) []float64 {
	if raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case []float64:
		return v
	case []any:
		result := make([]float64, len(v))
		for i, x := range v {
			switch f := x.(type) {
			case float64:
				result[i] = f
			case int:
				result[i] = float64(f)
			default:
				return nil
			}
		}
		return result
	default:
		rv := reflect.ValueOf(raw)
		if rv.Kind() == reflect.Slice {
			elemType := rv.Type().Elem()
			if elemType.Kind() == reflect.Struct {
				closeField, ok := elemType.FieldByName("Close")
				if ok && closeField.Type.Kind() == reflect.Float64 {
					n := rv.Len()
					result := make([]float64, n)
					for i := range n {
						result[i] = rv.Index(i).FieldByName("Close").Float()
					}
					return result
				}
			}
		}
		return nil
	}
}

// getStringParam extracts a string parameter with a default value.
func getStringParam(params map[string]any, key, defaultVal string) string {
	if params == nil {
		return defaultVal
	}
	if v, ok := params[key]; ok {
		switch val := v.(type) {
		case string:
			return val
		default:
			return fmt.Sprintf("%v", val)
		}
	}
	return defaultVal
}

// getFloatParam extracts a float64 parameter with a default value.
func getFloatParam(params map[string]any, key string, defaultVal float64) float64 {
	if params == nil {
		return defaultVal
	}
	if v, ok := params[key]; ok {
		switch val := v.(type) {
		case float64:
			return val
		case int:
			return float64(val)
		case int64:
			return float64(val)
		default:
			return defaultVal
		}
	}
	return defaultVal
}

// getIntParam extracts an int parameter with a default value.
func getIntParam(params map[string]any, key string, defaultVal int) int {
	if params == nil {
		return defaultVal
	}
	if v, ok := params[key]; ok {
		switch val := v.(type) {
		case int:
			return val
		case int64:
			return int(val)
		case float64:
			return int(val)
		default:
			return defaultVal
		}
	}
	return defaultVal
}
