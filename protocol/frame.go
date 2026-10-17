package protocol

import (
	"encoding/binary"
	"fmt"
	"io"
)

// Framing: Message, ReadMessage, WriteMessage, HeaderSize, MaxPayloadSize.

// MaxPayloadSize is the maximum allowed general payload size (64 KB).
const MaxPayloadSize = 65535

// MaxClientPayloadSize is the maximum allowed payload size for client messages (8 KB).
// Valid client messages (Identify, ChatSend, SetPresence, Pong) do not exceed ~4 KB.
const MaxClientPayloadSize = 8192

// HeaderSize is the 3-byte message header (type + length).
const HeaderSize = 3

// --- Raw message ---

// Message is a raw protocol message (header already parsed).
type Message struct {
	Type    byte
	Payload []byte
}

// ReadMessage reads a single framed message from r with default MaxPayloadSize.
func ReadMessage(r io.Reader) (Message, error) {
	return ReadMessageWithLimit(r, MaxPayloadSize)
}

// ReadMessageWithLimit reads a single framed message from r, enforcing maxLen before allocation.
func ReadMessageWithLimit(r io.Reader, maxLen uint32) (Message, error) {
	var header [HeaderSize]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return Message{}, err
	}

	msgType := header[0]
	length := uint32(binary.BigEndian.Uint16(header[1:3]))
	if length > maxLen {
		return Message{}, fmt.Errorf("payload size %d exceeds limit of %d bytes", length, maxLen)
	}

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

// --- Binary payload cursor helpers ---

type payloadWriter struct {
	buf []byte
}

func newWriter(capacity int) payloadWriter {
	return payloadWriter{buf: make([]byte, 0, capacity)}
}

func (w *payloadWriter) writeByte(v byte) {
	w.buf = append(w.buf, v)
}

func (w *payloadWriter) writeUint16(v uint16) {
	w.buf = binary.BigEndian.AppendUint16(w.buf, v)
}

func (w *payloadWriter) writeUint32(v uint32) {
	w.buf = binary.BigEndian.AppendUint32(w.buf, v)
}

func (w *payloadWriter) writeStr8(s string) {
	if len(s) > 255 {
		s = s[:255]
	}
	w.buf = append(w.buf, byte(len(s)))
	w.buf = append(w.buf, s...)
}

func (w *payloadWriter) writeStr16(s string) {
	if len(s) > MaxPayloadSize {
		s = s[:MaxPayloadSize]
	}
	w.buf = binary.BigEndian.AppendUint16(w.buf, uint16(len(s)))
	w.buf = append(w.buf, s...)
}

type payloadReader struct {
	data   []byte
	offset int
	err    error
}

func newReader(data []byte) payloadReader {
	return payloadReader{data: data}
}

func (r *payloadReader) next(n int) []byte {
	if r.err != nil {
		return nil
	}
	if r.offset+n > len(r.data) {
		r.err = fmt.Errorf("payload truncated")
		return nil
	}
	b := r.data[r.offset : r.offset+n]
	r.offset += n
	return b
}

func (r *payloadReader) readByte() byte {
	if b := r.next(1); b != nil {
		return b[0]
	}
	return 0
}

func (r *payloadReader) readUint16() uint16 {
	if b := r.next(2); b != nil {
		return binary.BigEndian.Uint16(b)
	}
	return 0
}

func (r *payloadReader) readUint32() uint32 {
	if b := r.next(4); b != nil {
		return binary.BigEndian.Uint32(b)
	}
	return 0
}

func (r *payloadReader) readStr8() string {
	if b := r.next(int(r.readByte())); b != nil {
		return string(b)
	}
	return ""
}

func (r *payloadReader) readStr16(maxLen int) string {
	n := int(r.readUint16())
	if r.err == nil && maxLen > 0 && n > maxLen {
		r.err = fmt.Errorf("chat message body exceeds limit: %d bytes (max %d)", n, maxLen)
	}
	if b := r.next(n); b != nil {
		return string(b)
	}
	return ""
}
