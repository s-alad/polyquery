package utils

// Truncate shortens a string to at most n runes, appending an ellipsis when truncated.
func Truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "…"
}

// TerminalLink returns an OSC 8 hyperlink for terminals that support it.
func TerminalLink(text, url string) string {
	const esc = "\x1b"
	return esc + "]8;;" + url + esc + "\\" + text + esc + "]8;;" + esc + "\\"
}
