package gui

import (
	"chatroom/internal/shared"
)

type EventHandler struct {
	client interface {
		SendMessage(*shared.Message) error
	}
}

func NewEventHandler(client interface{ SendMessage(*shared.Message) error }) *EventHandler {
	return &EventHandler{
		client: client,
	}
}

func (h *EventHandler) HandleSendMessage(content string) error {
	msg := &shared.Message{
		Type:    shared.TypePublic,
		Content: ConvertEmojis(content),
	}
	return h.client.SendMessage(msg)
}