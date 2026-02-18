package utils

import (
	"os"

	"golang.org/x/term"
)

func TerminalWidth() int {
	fd := int(os.Stdout.Fd())
	width, _, err := term.GetSize(fd)
	if err != nil || !term.IsTerminal(fd) || width <= 20 {
		return 0
	}
	return width
}
