package main

import (
	"os"

	"github.com/ajuda-dev/backend/src/config/database"
	"github.com/ajuda-dev/backend/src/config/job"
	"github.com/ajuda-dev/backend/src/config/logger"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	db, err := database.Connect()
	if err != nil {
		logger.Error("Failed to connect to the database", err)
		os.Exit(1)
	}
	cfg := job.ConfigFromEnv()
	job.StartPurgeJob(db, cfg)
	if !cfg.Enabled {
		return
	}
	select {}
}
