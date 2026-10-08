package main

import (
	"flag"
	"log/slog"
	"os"

	"github.com/nanatheblue/backtrack/server"
)

func main() {
	// Set up structured JSON logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	addr := flag.String("addr", ":9000", "TCP address to listen on")
	flag.Parse()

	srv, err := server.New(*addr)
	if err != nil {
		slog.Error("could not start server", "error", err)
		os.Exit(1)
	}
	defer srv.Close()

	err = srv.Run()
	if err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
