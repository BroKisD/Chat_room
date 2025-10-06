package shared

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

func ReadMessage(r io.Reader) (*Message, error) {
	reader := bufio.NewReader(r)
	line, err := reader.ReadString('\n')
	if err != nil {
		return nil, err
	}

	line = strings.TrimSpace(line)

	var msg Message
	if err := json.Unmarshal([]byte(line), &msg); err != nil {
		return nil, fmt.Errorf("json parse failed: %v", err)
	}

	return &msg, nil
}

func WriteMessage(w io.Writer, msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	data = append([]byte(string(data)), '\n')
	_, err = w.Write(data)
	return err
}
