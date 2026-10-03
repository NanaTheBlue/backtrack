package client

import (
	"fmt"
	"net"
	"os"
	"time"

	"github.com/nanatheblue/backtrack/protocol"
)

// simple test client

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
	payload, err := protocol.EncodeIdentify(username, avatarID)
	if err != nil {
		fmt.Printf("Failed to encode identify message: %v\n", err)
		return
	}

	// 2. Send the Identify message
	err = protocol.WriteMessage(conn, protocol.MsgIdentify, payload)
	if err != nil {
		fmt.Printf("Failed to send message: %v\n", err)
		return
	}

	fmt.Println("Message sent, waiting for response...")

	// 3. Read binary response frame
	response, err := protocol.ReadMessage(conn)
	if err != nil {
		fmt.Printf("Failed to read server response: %v\n", err)
		return
	}

	fmt.Printf("Received message type: 0x%02x, payload length: %d\n", response.Type, len(response.Payload))
}
