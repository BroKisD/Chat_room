package client

import (
	"chatroom/internal/client/networking"
	"chatroom/internal/shared"
)

type Client struct {
	conn     *networking.Connection
	username string
}

func New() *Client {
	return &Client{
		conn: networking.NewConnection(),
	}
}

func (c *Client) Connect(address string) error {
	if err := c.conn.Connect(address); err != nil {
		return err
	}

	// Send join message
	joinMsg := &shared.Message{
		Type:    shared.MessageTypeJoin,
		Sender:  c.username,
		Content: "joined the chat",
	}
	return c.conn.Send(joinMsg)
}

func (c *Client) Login(username string) error {
	c.username = username
	return nil
}

func (c *Client) SendMessage(content string) error {
	msg := &shared.Message{
		Type:    shared.MessageTypeChat,
		Sender:  c.username,
		Content: content,
	}
	return c.conn.Send(msg)
}

func (c *Client) Disconnect() error {
	leaveMsg := &shared.Message{
		Type:    shared.MessageTypeLeave,
		Sender:  c.username,
		Content: "left the chat",
	}
	if err := c.conn.Send(leaveMsg); err != nil {
		return err
	}
	return c.conn.Close()
}
