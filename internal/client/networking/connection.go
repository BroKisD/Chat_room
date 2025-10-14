package networking

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"time"

	"chatroom/internal/shared"
)

type Connection struct {
	address  string
	conn     net.Conn
	incoming chan *shared.Message
	isClosed bool
}

func NewConnection() *Connection {
	return &Connection{
		incoming: make(chan *shared.Message, 100),
	}
}

func (c *Connection) Connect(address string) error {
	conn, err := net.Dial("tcp", address)
	if err != nil {
		return err
	}
	c.conn = conn
	c.address = address
	c.isClosed = false

	go c.listen()
	return nil
}

func (c *Connection) listen() {
	reader := bufio.NewReader(c.conn)
	for {
		if c.isClosed {
			return
		}

		msg, err := shared.ReadMessage(reader)
		if err != nil {
			if err == io.EOF {
				fmt.Println("[INFO] Server closed connection")
			} else {
				fmt.Println("[WARN] Read error:", err)
			}

			close(c.incoming)
			return
		}

		select {
		case c.incoming <- msg:
		default:
			fmt.Println("[WARN] Incoming channel full, dropping message")
		}
	}
}

func (c *Connection) Send(msg *shared.Message) error {
	if c.conn == nil {
		return fmt.Errorf("connection inactive")
	}
	return shared.WriteMessage(c.conn, msg)
}

func (c *Connection) Incoming() <-chan *shared.Message {
	return c.incoming
}

func (c *Connection) Close() error {
	c.isClosed = true
	if c.conn != nil {
		err := c.conn.Close()
		c.conn = nil
		return err
	}
	return nil
}

func (c *Connection) Reconnect() error {
	if c.address == "" {
		return fmt.Errorf("no known address to reconnect")
	}

	for {
		fmt.Println("[INFO] Attempting to reconnect to", c.address)
		conn, err := net.Dial("tcp", c.address)
		if err == nil {
			fmt.Println("[INFO] Reconnected successfully")

			c.conn = conn
			c.isClosed = false
			c.incoming = make(chan *shared.Message, 100)

			go c.listen()
			return nil
		}

		fmt.Println("[ERROR] Reconnect failed:", err)
		time.Sleep(5 * time.Second)
	}
}
