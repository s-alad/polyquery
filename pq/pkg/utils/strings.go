package utils

// Truncate shortens a string to at most n runes, appending an ellipsis when truncated.
func Truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}
