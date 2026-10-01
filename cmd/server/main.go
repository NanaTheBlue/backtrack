package main

import (
	"flag"
	"log"
	"time"

	"github.com/skyler/backtrack/client"
	"github.com/skyler/backtrack/server"
)

func main() {
	addr := flag.String("addr", ":9000", "TCP address to listen on")
	flag.Parse()

	srv, err := server.New(*addr)
	if err != nil {
		log.Fatalf("could not start server: %v", err)
	}
	defer srv.Close()

	go srv.Run() // blocks forever

	time.Sleep(50 * time.Millisecond)

	log.Println("Starting client...")
	client.RunClient()
}
