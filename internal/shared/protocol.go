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

	// Decrypt first
	decrypted, err := Decrypt(line)
	if err != nil {
		return nil, fmt.Errorf("decrypt failed: %v", err)
	}

	var msg Message
	if err := json.Unmarshal([]byte(decrypted), &msg); err != nil {
		return nil, fmt.Errorf("json parse failed: %v", err)
	}

	return &msg, nil
}

func WriteMessage(w io.Writer, msg *Message) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	// Encrypt JSON before sending
	encrypted, err := Encrypt(string(data))
	if err != nil {
		return err
	}

	// Write encrypted message with newline
	data = append([]byte(encrypted), '\n')
	_, err = w.Write(data)
	return err
}
