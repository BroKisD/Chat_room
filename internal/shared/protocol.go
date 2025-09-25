package shared

import (
	"encoding/json"
	"io"
)

func ReadMessage(r io.Reader) (*Message, error) {
	decoder := json.NewDecoder(r)
	var msg Message
	if err := decoder.Decode(&msg); err != nil {
		return nil, err
	}
	return &msg, nil
}

func WriteMessage(w io.Writer, msg *Message) error {
	encoder := json.NewEncoder(w)
	return encoder.Encode(msg)
}
