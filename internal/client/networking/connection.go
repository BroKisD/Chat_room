package networking

import (
	"bufio"
	"fmt"
	"io"
	"net"

	"chatroom/internal/shared"
)

type Connection struct {
	conn     net.Conn
	incoming chan *shared.Message
}

func NewConnection() *Connection {
	return &Connection{
		incoming: make(chan *shared.Message, 100), // buffer để tránh nghẽn
	}
}

func (c *Connection) Connect(address string) error {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return err
	}
	c.conn = conn

	// bắt đầu goroutine đọc message
	go c.listen()

	return nil
}

func (c *Connection) listen() {
	reader := bufio.NewReader(c.conn)
	for {
		msg, err := shared.ReadMessage(reader)
		if err != nil {
			if err == io.EOF {
				fmt.Println("[INFO] Server closed connection")
				close(c.incoming)
				return
			}
			fmt.Println("[WARN] Read error:", err)
			continue
		}

		fmt.Println("[DEBUG] Received from server:", msg)
		c.incoming <- msg
	}
}

func (c *Connection) Send(msg *shared.Message) error {
	return shared.WriteMessage(c.conn, msg)
}

func (c *Connection) Incoming() <-chan *shared.Message {
	return c.incoming
}

func (c *Connection) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
