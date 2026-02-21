package logging

import "fmt"

// Color represents a terminal ANSI color escape code.
type Color = string

// Reset code
const (
	Reset Color = "\033[0m"
)

// Foreground colors
const (
	Black   Color = "\033[30m"
	Red     Color = "\033[31m"
	Green   Color = "\033[32m"
	Yellow  Color = "\033[33m"
	Blue    Color = "\033[34m"
	Purple  Color = "\033[35m"
	Cyan    Color = "\033[36m"
	White   Color = "\033[37m"
	Gray    Color = "\033[90m"
	Default Color = "\033[39m"
)

// Bold foreground colors
const (
	BoldBlack   Color = "\033[1;30m"
	BoldRed     Color = "\033[1;31m"
	BoldGreen   Color = "\033[1;32m"
	BoldYellow  Color = "\033[1;33m"
	BoldBlue    Color = "\033[1;34m"
	BoldPurple  Color = "\033[1;35m"
	BoldCyan    Color = "\033[1;36m"
	BoldWhite   Color = "\033[1;37m"
	BoldGray    Color = "\033[1;90m"
	BoldDefault Color = "\033[1;39m"
)

// Background colors
const (
	BgBlack   Color = "\033[40m"
	BgRed     Color = "\033[41m"
	BgGreen   Color = "\033[42m"
	BgYellow  Color = "\033[43m"
	BgBlue    Color = "\033[44m"
	BgPurple  Color = "\033[45m"
	BgCyan    Color = "\033[46m"
	BgWhite   Color = "\033[47m"
	BgGray    Color = "\033[100m"
	BgDefault Color = "\033[49m"
)

// Bright background colors (high intensity)
const (
	BgBrightBlack  Color = "\033[100m"
	BgBrightRed    Color = "\033[101m"
	BgBrightGreen  Color = "\033[102m"
	BgBrightYellow Color = "\033[103m"
	BgBrightBlue   Color = "\033[104m"
	BgBrightPurple Color = "\033[105m"
	BgBrightCyan   Color = "\033[106m"
	BgBrightWhite  Color = "\033[107m"
)

// Text styles
const (
	Bold      Color = "\033[1m"
	Dim       Color = "\033[2m"
	Italic    Color = "\033[3m"
	Underline Color = "\033[4m"
	Blink     Color = "\033[5m"
	Reverse   Color = "\033[7m"
	Hidden    Color = "\033[8m"
	Strike    Color = "\033[9m"
)

// Colorize wraps text with the given color and reset code.
func Colorize(color Color, text string) string {
	return color + text + Reset
}

// Colorizef wraps formatted text with the given color.
func Colorizef(color Color, format string, args ...any) string {
	return color + fmt.Sprintf(format, args...) + Reset
}

// Combine combines multiple colors/styles into one.
// Example: Combine(BoldWhite, BgRed) for bold white text on red background.
func Combine(colors ...Color) Color {
	var result Color
	for _, c := range colors {
		result += c
	}
	return result
}

// Styled applies combined colors to text and resets.
// Example: Styled("ERROR", BoldWhite, BgRed) -> bold white "ERROR" on red background.
func Styled(text string, colors ...Color) string {
	return Combine(colors...) + text + Reset
}

// Styledf applies combined colors to formatted text and resets.
func Styledf(format string, colors []Color, args ...any) string {
	return Combine(colors...) + fmt.Sprintf(format, args...) + Reset
}

// Pad returns text padded to specified width with color applied.
func Pad(color Color, text string, width int) string {
	return Colorizef(color, "%-*s", width, text)
}

// PadLeft returns text left-padded to specified width with color applied.
func PadLeft(color Color, text string, width int) string {
	return Colorizef(color, "%*s", width, text)
}
