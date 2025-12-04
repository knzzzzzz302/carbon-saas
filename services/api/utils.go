package main

import (
	"encoding/json"
	"fmt"
)

func toStringID(v interface{}) string {
	switch val := v.(type) {
	case float64:
		return fmt.Sprintf("%.0f", val)
	case int64:
		return fmt.Sprintf("%d", val)
	case int:
		return fmt.Sprintf("%d", val)
	case string:
		return val
	default:
		return fmt.Sprint(val)
	}
}

func toJSONB(m map[string]string) string {
	if m == nil {
		return "{}"
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "{}"
	}
	return string(b)
}


