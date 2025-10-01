package networking

import (
	"net"

	"chatroom/internal/shared"
)

type Connection struct {
	conn net.Conn
}

func NewConnection() *Connection {
	return &Connection{}
}

func (c *Connection) Connect(address string) error {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return err
	}
	c.conn = conn
	return nil
}

func (c *Connection) Send(msg *shared.Message) error {
	return shared.WriteMessage(c.conn, msg)
}

func (c *Connection) Receive() (*shared.Message, error) {
	return shared.ReadMessage(c.conn)
}

func (c *Connection) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
