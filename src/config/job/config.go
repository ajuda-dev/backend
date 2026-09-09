package job

import (
	"os"
	"strconv"
	"strings"
)

func ConfigFromEnv() PurgeConfig {
	return PurgeConfig{
		Enabled:       envBool("PURGE_ENABLED", false),
		OlderThanDays: envInt("PURGE_OLDER_THAN_DAYS", 30),
		IntervalHours: envInt("PURGE_INTERVAL_HOURS", 24),
	}
}

func envBool(key string, fallback bool) bool {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseBool(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func envInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
