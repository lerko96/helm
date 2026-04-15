package handlers

import "strings"

// validEntityTypes is the whitelist of entity types accepted by polymorphic endpoints
// (tags, attachments, reminders).
var validEntityTypes = map[string]bool{
	"note":      true,
	"todo":      true,
	"memo":      true,
	"bookmark":  true,
	"clipboard": true,
}

// validEntityType returns true if the entity type is in the whitelist.
func validEntityType(entityType string) bool {
	return validEntityTypes[entityType]
}

// sanitizeFTSQuery wraps a user-supplied search string in FTS5 phrase quotes and
// appends a prefix wildcard. Internal double-quotes are escaped by doubling them.
// This prevents FTS5 syntax errors from special characters in user input.
func sanitizeFTSQuery(s string) string {
	s = strings.ReplaceAll(s, `"`, `""`)
	return `"` + s + `"*`
}
