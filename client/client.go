package client

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"os"
	"time"
)

// simple test client

const MsgIdentify byte = 0x01

type Frame struct {
	Type    byte
	Payload []byte
}

func RunClient() {
	Run("localhost:9000", "bingus", 123)
}

func Run(address string, username string, avatarID uint16) {
	conn, err := net.DialTimeout("tcp", address, 5*time.Second)
	if err != nil {
		fmt.Printf("Failed to connect to server: %v\n", err)
		os.Exit(1)
	}
	defer conn.Close()

	fmt.Printf("Connected to TCP server at %s\n", address)
	defer fmt.Println("Client exiting...")

	err = conn.SetDeadline(time.Now().Add(10 * time.Second))
	if err != nil {
		fmt.Printf("Failed to set connection deadline: %v\n", err)
		return
	}

	// 1. Encode the Identify payload
	payload, err := EncodeIdentify(username, avatarID)
	if err != nil {
		fmt.Printf("Failed to encode identify message: %v\n", err)
		return
	}

	// 2. Frame the message (Type + Length + Payload) & send
	packet := frameMessage(MsgIdentify, payload)
	err = sendMessage(conn, packet)
	if err != nil {
		fmt.Printf("Failed to send message: %v\n", err)
		return
	}

	fmt.Println("Message sent, waiting for response...")

	// 3. Read binary response frame
	response, err := readFrame(conn)
	if err != nil {
		fmt.Printf("Failed to read server response: %v\n", err)
		return
	}

	fmt.Printf("Received message type: 0x%02x, payload length: %d\n", response.Type, len(response.Payload))
}

func sendMessage(conn net.Conn, m []byte) error {
	_, err := conn.Write(m)
	if err != nil {
		return fmt.Errorf("failed to send message: %v", err)
	}
	fmt.Printf("Sent: %v", m)
	return nil
}

func EncodeIdentify(username string, avatarID uint16) ([]byte, error) {
	nameLen := len(username)
	if nameLen > 255 {
		return nil, fmt.Errorf("username too long")
	}

	// Payload layout: nameLen (1) + username (N) + avatarID (2)
	payload := make([]byte, 1+nameLen+2)
	payload[0] = byte(nameLen)
	copy(payload[1:], username)
	binary.BigEndian.PutUint16(payload[1+nameLen:], avatarID)

	return payload, nil
}

func frameMessage(msgType byte, payload []byte) []byte {
	length := uint16(len(payload))
	frame := make([]byte, 1+2+len(payload))
	frame[0] = msgType
	binary.BigEndian.PutUint16(frame[1:3], length)
	copy(frame[3:], payload)

	return frame
}

func readFrame(conn net.Conn) (Frame, error) {
	// 1. Read the 3-byte frame header (1 byte type + 2 bytes len)
	header := make([]byte, 3)
	_, err := io.ReadFull(conn, header)
	if err != nil {
		return Frame{}, err
	}

	msgType := header[0]
	payloadLen := binary.BigEndian.Uint16(header[1:3])

	// 2. Read the exact number of payload bytes
	payload := make([]byte, payloadLen)
	_, err = io.ReadFull(conn, payload)
	if err != nil {
		return Frame{}, err
	}

	return Frame{Type: msgType, Payload: payload}, nil
}
