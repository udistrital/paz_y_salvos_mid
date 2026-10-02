package helpers

import "strings"

func ContainsIgnoreCase(value, substring string) bool {
	return strings.Contains(strings.ToLower(value), strings.ToLower(substring))
}
