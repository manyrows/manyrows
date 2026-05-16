package validation

import (
	"strings"
)

func LooksLikeUniqueViolation(err error) bool {
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "duplicate key") {
		return true
	}
	if strings.Contains(msg, "unique") && strings.Contains(msg, "constraint") {
		return true
	}
	if strings.Contains(msg, "already exists") && strings.Contains(msg, "slug") {
		return true
	}
	return false
}
