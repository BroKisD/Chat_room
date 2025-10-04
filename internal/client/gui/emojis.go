package gui

import (
	"strings"
)

var emojiMap = map[string]string{
	":)":     "😊",
	"<3":     "❤️",
	":like:": "👍",
	"XD":     "😂",
	":(":     "😢",
}

func ConvertEmojis(text string) string {
	for code, emoji := range emojiMap {
		text = strings.ReplaceAll(text, code, emoji)
	}
	return text
}
