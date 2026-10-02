package main

import (
	"flag"

	"github.com/nanatheblue/backtrack/client"
)

func main() {
	addr := flag.String("addr", "localhost:9000", "TCP server address to connect to")
	username := flag.String("user", "bingus", "username to identify with")
	avatarID := flag.Uint("avatar", 123, "avatar ID")
	flag.Parse()

	client.Run(*addr, *username, uint16(*avatarID))
}
