package mapper

import "fmt"

// StringifyKeys converts keys to strings
func StringifyKeys(val interface{}) interface{} {
	switch v := val.(type) {
	case []interface{}:
		for n, item := range v {
			v[n] = StringifyKeys(item)
		}
	case map[interface{}]interface{}:
		m := make(map[string]interface{})
		for key, value := range v {
			m[fmt.Sprintf("%v", key)] = StringifyKeys(value)
		}
		val = m
	case map[string]interface{}:
		for key, value := range v {
			v[key] = StringifyKeys(value)
		}
	}
	return val
}
