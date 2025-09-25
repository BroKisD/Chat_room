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
	return c.conn.Connect(address)
}

func (c *Client) Login(username, password string) error {
	// Login implementation
	return nil
}

func (c *Client) SendMessage(msg *shared.Message) error {
	return c.conn.Send(msg)
}
