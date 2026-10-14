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
	dbURL := flag.String("db", os.Getenv("DATABASE_URL"),
		"PostgreSQL connection string (defaults to $DATABASE_URL); leave empty to run without message persistence")
	flag.Parse()

	// Persistence is optional: without a database the server runs fully in memory.
	var repos []message.Repository
	if *dbURL != "" {
		dbpool, err := repository.NewPostgresPool(context.Background(), *dbURL)
		if err != nil {
			slog.Error("failed to connect to database", "error", err)
			os.Exit(1)
		}
		defer dbpool.Close()
		repos = append(repos, message.New(dbpool))
		slog.Info("message persistence enabled")
	} else {
		slog.Warn("DATABASE_URL not set; running without message persistence")
	}

	srv, err := server.New(*addr, repos...)
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
