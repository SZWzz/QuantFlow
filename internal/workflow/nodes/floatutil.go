package nodes

import "reflect"

// extractFloat64Slice converts any slice-like value to []float64.
// Supports: []float64, []any, and any []struct with a float64 Close field (e.g., market.OHLCVBar).
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
		// Check for []struct with a float64 Close field via reflection.
		// Works even for empty slices by inspecting the type's element struct field.
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
