package client

import (
	"fmt"
	"strings"
	"time"

	"chatroom/internal/client/networking"
	"chatroom/internal/shared"
)

type Client struct {
	conn        *networking.Connection
	username    string
	activeUsers []string
	onMessage   func(msg string)
}

func New() *Client {
	return &Client{
		conn: networking.NewConnection(),
	}
}

func (c *Client) SetMessageHandler(handler func(msg string)) {
	c.onMessage = handler
}

func (c *Client) Connect(address string) error {
	if err := c.conn.Connect(address); err != nil {
		return err
	}

	// Send authentication message
	authMsg := &shared.Message{
		Type:    shared.TypeAuth,
		From:    c.username,
		Content: "auth",
	}

	if err := c.conn.Send(authMsg); err != nil {
		return fmt.Errorf("auth failed: %v", err)
	}

	authResp := <-c.conn.Incoming()
	if authResp.Type != shared.TypeAuthResponse {
		return fmt.Errorf("unexpected response type: %s", authResp.Type)
	}
	if !authResp.Success {
		return fmt.Errorf("authentication failed: %s", authResp.Error)
	}

	// Start message listener
	go c.handleMessages()

	return nil
}

func (c *Client) Login(username string) error {
	c.username = username
	return nil
}

func (c *Client) SendMessage(content string) error {
	msg := &shared.Message{
		Type:      shared.TypePublic,
		From:      c.username,
		Content:   content,
		Timestamp: time.Now(),
	}
	return c.conn.Send(msg)
}

func (c *Client) SendPrivateMessage(target, content string) error {
	msg := &shared.Message{
		Type:      shared.TypePrivate,
		From:      c.username,
		To:        target,
		Content:   content,
		Timestamp: time.Now(),
	}
	return c.conn.Send(msg)
}

func (c *Client) Disconnect() error {
	msg := &shared.Message{
		Type:      shared.TypeLeave,
		From:      c.username,
		Timestamp: time.Now(),
	}
	if err := c.conn.Send(msg); err != nil {
		return err
	}
	return c.conn.Close()
}

func (c *Client) GetActiveUsers() []string {
	return c.activeUsers
}

func (c *Client) handleMessages() {
	for msg := range c.conn.Incoming() {
		switch msg.Type {
		case shared.TypePublic:
			c.formatAndDisplayMessage(msg)
		case shared.TypePrivate:
			c.formatAndDisplayPrivateMessage(msg)
		case shared.TypeUserList:
			c.activeUsers = msg.Users
			c.notifyUserListUpdate()
		case shared.TypeJoin, shared.TypeLeave:
			c.displaySystemMessage(msg)
		case shared.TypeError:
			c.displayErrorMessage(msg)
		}
	}
}

func (c *Client) formatAndDisplayMessage(msg *shared.Message) {
	formatted := fmt.Sprintf("(Global) (%s) %s: %s",
		msg.Timestamp.Format("15:04:05"),
		msg.From,
		msg.Content)
	c.displayMessage(formatted)
}

func (c *Client) formatAndDisplayPrivateMessage(msg *shared.Message) {
	formatted := fmt.Sprintf("(Private) (%s) %s: %s",
		msg.Timestamp.Format("15:04:05"),
		msg.From,
		msg.Content)
	c.displayMessage(formatted)
}

func (c *Client) displaySystemMessage(msg *shared.Message) {
	formatted := fmt.Sprintf("(System) (%s) %s",
		msg.Timestamp.Format("15:04:05"),
		msg.Content)
	c.displayMessage(formatted)
}

func (c *Client) displayErrorMessage(msg *shared.Message) {
	formatted := fmt.Sprintf("(Error) %s", msg.Content)
	c.displayMessage(formatted)
}

func (c *Client) notifyUserListUpdate() {
	formatted := fmt.Sprintf("Active users: %s",
		strings.Join(c.activeUsers, ", "))
	c.displayMessage(formatted)
}

func (c *Client) displayMessage(msg string) {
	if c.onMessage != nil {
		c.onMessage(msg)
	} else {
		fmt.Println(msg)
	}
}
