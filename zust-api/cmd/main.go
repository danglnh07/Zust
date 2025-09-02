package main

import (
	"database/sql"
	"log/slog"
	"os"
	"zust/api"
	"zust/service/security"

	_ "github.com/lib/pq"
)

func main() {
	// Initialize logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Load config from .env
	err := security.LoadConfig("./.env")
	if err != nil {
		logger.Error("Failed to load configurations from .env", "error", err)
		return
	}
	config := security.GetConfig()

	// Connect to database
	conn, err := sql.Open(config.DbDriver, config.DbSource)
	if err != nil {
		logger.Error("Error ebstablish database connection", "error", err)
		return
	}

	// Create and start server
	svr := api.NewServer(conn, &config, logger)
	if err := svr.Start(); err != nil {
		logger.Error("Error: server unexpectedly shutdown", "error", err)
	}
}
