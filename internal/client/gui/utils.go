package gui

import "strings"

func ConvertEmojis(text string) string {
	for _, group := range emojiMap {
		for code, emoji := range group {
			text = strings.ReplaceAll(text, code, emoji)
		}
	}
	return text
}