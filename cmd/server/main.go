package main

import (
	"flag"
	"log"

	"github.com/nanatheblue/backtrack/server"
)

func main() {
	addr := flag.String("addr", ":9000", "TCP address to listen on")
	flag.Parse()

	srv, err := server.New(*addr)
	if err != nil {
		log.Fatalf("could not start server: %v", err)
	}
	defer srv.Close()

	err = srv.Run()
	if err != nil {
		log.Fatalf("server error: %v", err)
	}
}
