package protocol

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
