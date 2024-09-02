package core

import (
	"fmt"
)

// ColorMap maps color names to ANSI color codes.
var ColorMap = map[string]string{
	"black":   "30",
	"red":     "31",
	"green":   "32",
	"yellow":  "33",
	"blue":    "34",
	"magenta": "35",
	"cyan":    "36",
	"white":   "37",
}

// ApplyColor applies the specified color to the text if the color exists in ColorMap.
func ApplyColor(colorName, text string) string {

	Logger.Debug("Applying color to text", "colorName", colorName, "text", text)

	if colorCode, exists := ColorMap[colorName]; exists {
		return fmt.Sprintf("\033[%sm%s\033[0m", colorCode, text)
	}
	// Return the original text if colorName is not found
	return text
}
