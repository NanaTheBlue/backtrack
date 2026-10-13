package main

import (
	"context"
	"flag"
	"log/slog"
	"os"

	repository "github.com/nanatheblue/backtrack/repository/db"
	"github.com/nanatheblue/backtrack/repository/message"
	"github.com/nanatheblue/backtrack/server"
)

func main() {
	// Set up structured JSON logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	addr := flag.String("addr", ":9000", "TCP address to listen on")
	flag.Parse()

	ctx := context.Background()
	dbpool, err := repository.NewPostgresPool(ctx, "")
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer dbpool.Close()

	msgRepo := message.New(dbpool)

	srv, err := server.New(*addr, msgRepo)
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
