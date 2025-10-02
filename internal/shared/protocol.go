package shared

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
)

func ReadMessage(reader *bufio.Reader) (*Message, error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			return nil, err
		}
		fmt.Println("[ERROR] Read from server:", err)
		return nil, err
	}

	var msg Message
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		fmt.Println("[ERROR] Failed to unmarshal JSON:", err, "Line:", line)
		return nil, err
	}

	return &msg, nil
}

func WriteMessage(w io.Writer, msg *Message) error {
	// Convert to JSON
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Add newline and write
	data = append(data, '\n')
	_, err = w.Write(data)
	return err
}
