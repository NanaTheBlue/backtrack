package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

// Framing: Message, ReadMessage, WriteMessage, HeaderSize, MaxPayloadSize.

// MaxPayloadSize is the maximum allowed payload size (64 KB).
const MaxPayloadSize = 65535

// HeaderSize is the 3-byte message header (type + length).
const HeaderSize = 3

// --- Raw message ---

// Message is a raw protocol message (header already parsed).
type Message struct {
	Type    byte
	Payload []byte
}

// ReadMessage reads a single framed message from r.
func ReadMessage(r io.Reader) (Message, error) {
	var header [HeaderSize]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return Message{}, err
	}

	msgType := header[0]
	length := binary.BigEndian.Uint16(header[1:3])

	payload := make([]byte, length)
	if length > 0 {
		if _, err := io.ReadFull(r, payload); err != nil {
			return Message{}, fmt.Errorf("short payload read: %w", err)
		}
	}

	return Message{Type: msgType, Payload: payload}, nil
}

// WriteMessage writes a framed message to w.
func WriteMessage(w io.Writer, msgType byte, payload []byte) error {
	if len(payload) > MaxPayloadSize {
		return fmt.Errorf("payload too large: %d bytes (max %d)", len(payload), MaxPayloadSize)
	}

	var header [HeaderSize]byte
	header[0] = msgType
	binary.BigEndian.PutUint16(header[1:3], uint16(len(payload)))

	if _, err := w.Write(header[:]); err != nil {
		return err
	}
	if len(payload) > 0 {
		if _, err := w.Write(payload); err != nil {
			return err
		}
	}
	return nil
}
