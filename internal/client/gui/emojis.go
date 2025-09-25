package gui

import (
	"strings"
)

var emojiMap = map[string]string{
	":smile:":    "😊",
	":heart:":    "❤️",
	":thumbsup:": "👍",
	// Add more emoji mappings
}

func ConvertEmojis(text string) string {
	for code, emoji := range emojiMap {
		text = strings.ReplaceAll(text, code, emoji)
	}
	return text
}
