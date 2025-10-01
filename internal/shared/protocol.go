package shared

import (
	"bufio"
	"encoding/json"
	"io"
)

func ReadMessage(r io.Reader) (*Message, error) {
	reader := bufio.NewReader(r)

	// Read one line (until \n)
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	// Parse JSON
	var msg Message
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
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
