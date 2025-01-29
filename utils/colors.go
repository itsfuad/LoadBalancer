package utils

type Color string

const (
	RED    Color = "\033[31m"
	BLUE   Color = "\033[34m"
	GREEN  Color = "\033[32m"
	YELLOW Color = "\033[33m"
	RESET  Color = "\033[0m"
)

// Colorize returns the string s wrapped in the ANSI color c
func Colorize(s string, c Color) string {
	return string(c) + s + string(RESET)
}
