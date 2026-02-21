package logging

import (
	"net/http"
	"time"
)

// ColorScheme maps semantic elements to colors.
// Implement this interface to fully customize color behavior.
type ColorScheme interface {
	// StatusColor returns the color for an HTTP status code.
	StatusColor(code int) Color
	// MethodColor returns the color for an HTTP method.
	MethodColor(method string) Color
	// DurationColor returns the color based on request duration.
	DurationColor(d time.Duration) Color
	// LevelColor returns the color for a log level.
	LevelColor(level string) Color
}

// DefaultColorScheme provides configurable color mappings with sensible defaults.
// Override individual fields to customize specific colors.
// Zero values fall back to sensible defaults.
type DefaultColorScheme struct {
	// Status code colors (foreground + optional background)
	Status1xx Color // Informational
	Status2xx Color // Success
	Status3xx Color // Redirection
	Status4xx Color // Client Error
	Status5xx Color // Server Error

	// HTTP method colors
	MethodGET     Color
	MethodPOST    Color
	MethodPUT     Color
	MethodDELETE  Color
	MethodPATCH   Color
	MethodHEAD    Color
	MethodOPTIONS Color

	// Duration thresholds and colors
	DurationFast          Color         // < FastThreshold
	DurationMedium        Color         // FastThreshold <= d < SlowThreshold
	DurationSlow          Color         // >= SlowThreshold
	DurationFastThreshold time.Duration // Default: 100ms
	DurationSlowThreshold time.Duration // Default: 500ms

	// Log level colors
	LevelDebug Color
	LevelInfo  Color
	LevelWarn  Color
	LevelError Color
	LevelFatal Color
}

// NewDefaultColorScheme returns a scheme with sensible defaults.
// All colors use foreground only. Customize fields for background colors.
func NewDefaultColorScheme() *DefaultColorScheme {
	return &DefaultColorScheme{
		// Status codes - foreground colors
		Status1xx: Cyan,
		Status2xx: Green,
		Status3xx: Cyan,
		Status4xx: Yellow,
		Status5xx: Red,

		// HTTP methods
		MethodGET:     Blue,
		MethodPOST:    Cyan,
		MethodPUT:     Yellow,
		MethodDELETE:  Red,
		MethodPATCH:   Purple,
		MethodHEAD:    White,
		MethodOPTIONS: Gray,

		// Duration
		DurationFast:          Green,
		DurationMedium:        Yellow,
		DurationSlow:          Red,
		DurationFastThreshold: 100 * time.Millisecond,
		DurationSlowThreshold: 500 * time.Millisecond,

		// Log levels
		LevelDebug: Gray,
		LevelInfo:  Green,
		LevelWarn:  Yellow,
		LevelError: Red,
		LevelFatal: Combine(BoldWhite, BgRed),
	}
}

// NewBoldColorScheme returns a scheme with bold foreground colors.
func NewBoldColorScheme() *DefaultColorScheme {
	return &DefaultColorScheme{
		Status1xx: BoldCyan,
		Status2xx: BoldGreen,
		Status3xx: BoldCyan,
		Status4xx: BoldYellow,
		Status5xx: BoldRed,

		MethodGET:     BoldBlue,
		MethodPOST:    BoldCyan,
		MethodPUT:     BoldYellow,
		MethodDELETE:  BoldRed,
		MethodPATCH:   BoldPurple,
		MethodHEAD:    BoldWhite,
		MethodOPTIONS: Gray,

		DurationFast:          BoldGreen,
		DurationMedium:        BoldYellow,
		DurationSlow:          BoldRed,
		DurationFastThreshold: 100 * time.Millisecond,
		DurationSlowThreshold: 500 * time.Millisecond,

		LevelDebug: Gray,
		LevelInfo:  BoldGreen,
		LevelWarn:  BoldYellow,
		LevelError: BoldRed,
		LevelFatal: Combine(BoldWhite, BgRed),
	}
}

// NewBackgroundColorScheme returns a scheme using background colors for emphasis.
// Great for status codes and log levels that need to stand out.
func NewBackgroundColorScheme() *DefaultColorScheme {
	return &DefaultColorScheme{
		// Status codes with background colors
		Status1xx: Combine(Black, BgCyan),
		Status2xx: Combine(Black, BgGreen),
		Status3xx: Combine(Black, BgCyan),
		Status4xx: Combine(Black, BgYellow),
		Status5xx: Combine(BoldWhite, BgRed),

		// Methods stay foreground (less noisy)
		MethodGET:     BoldBlue,
		MethodPOST:    BoldCyan,
		MethodPUT:     BoldYellow,
		MethodDELETE:  BoldRed,
		MethodPATCH:   BoldPurple,
		MethodHEAD:    BoldWhite,
		MethodOPTIONS: Gray,

		DurationFast:          Green,
		DurationMedium:        Yellow,
		DurationSlow:          Combine(BoldWhite, BgRed),
		DurationFastThreshold: 100 * time.Millisecond,
		DurationSlowThreshold: 500 * time.Millisecond,

		// Log levels with background
		LevelDebug: Gray,
		LevelInfo:  Combine(Black, BgGreen),
		LevelWarn:  Combine(Black, BgYellow),
		LevelError: Combine(BoldWhite, BgRed),
		LevelFatal: Combine(BoldWhite, BgRed, Blink),
	}
}

// StatusColor returns the color for an HTTP status code.
func (s *DefaultColorScheme) StatusColor(code int) Color {
	switch {
	case code >= 100 && code < 200:
		return s.withDefault(s.Status1xx, Cyan)
	case code >= 200 && code < 300:
		return s.withDefault(s.Status2xx, Green)
	case code >= 300 && code < 400:
		return s.withDefault(s.Status3xx, Cyan)
	case code >= 400 && code < 500:
		return s.withDefault(s.Status4xx, Yellow)
	case code >= 500:
		return s.withDefault(s.Status5xx, Red)
	default:
		return White
	}
}

// MethodColor returns the color for an HTTP method.
func (s *DefaultColorScheme) MethodColor(method string) Color {
	switch method {
	case http.MethodGet:
		return s.withDefault(s.MethodGET, Blue)
	case http.MethodPost:
		return s.withDefault(s.MethodPOST, Cyan)
	case http.MethodPut:
		return s.withDefault(s.MethodPUT, Yellow)
	case http.MethodDelete:
		return s.withDefault(s.MethodDELETE, Red)
	case http.MethodPatch:
		return s.withDefault(s.MethodPATCH, Purple)
	case http.MethodHead:
		return s.withDefault(s.MethodHEAD, White)
	case http.MethodOptions:
		return s.withDefault(s.MethodOPTIONS, Gray)
	default:
		return White
	}
}

// DurationColor returns the color based on request duration.
func (s *DefaultColorScheme) DurationColor(d time.Duration) Color {
	fastThreshold := s.DurationFastThreshold
	if fastThreshold == 0 {
		fastThreshold = 100 * time.Millisecond
	}
	slowThreshold := s.DurationSlowThreshold
	if slowThreshold == 0 {
		slowThreshold = 500 * time.Millisecond
	}

	switch {
	case d < fastThreshold:
		return s.withDefault(s.DurationFast, Green)
	case d < slowThreshold:
		return s.withDefault(s.DurationMedium, Yellow)
	default:
		return s.withDefault(s.DurationSlow, Red)
	}
}

// LevelColor returns the color for a log level.
func (s *DefaultColorScheme) LevelColor(level string) Color {
	switch level {
	case "debug", "DEBUG":
		return s.withDefault(s.LevelDebug, Gray)
	case "info", "INFO":
		return s.withDefault(s.LevelInfo, Green)
	case "warn", "WARN", "warning", "WARNING":
		return s.withDefault(s.LevelWarn, Yellow)
	case "error", "ERROR":
		return s.withDefault(s.LevelError, Red)
	case "fatal", "FATAL", "panic", "PANIC":
		return s.withDefault(s.LevelFatal, Combine(BoldWhite, BgRed))
	default:
		return White
	}
}

// withDefault returns the value if not empty, otherwise returns the default.
func (s *DefaultColorScheme) withDefault(value, defaultValue Color) Color {
	if value == "" {
		return defaultValue
	}
	return value
}

// Ensure DefaultColorScheme implements ColorScheme.
var _ ColorScheme = (*DefaultColorScheme)(nil)
