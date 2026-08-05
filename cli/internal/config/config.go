package config

import "os"

type Config struct {
	APIBaseURL string
	AgentToken string
}

func Load() Config {
	base := os.Getenv("ENVO_API_URL")
	if base == "" {
		// Default to the hosted Envo API for production users.
		// Self-hosted users can override with ENVO_API_URL or --api.
		base = "https://api.envo.scopophobic.xyz"
	}

	return Config{
		APIBaseURL: base,
		AgentToken: os.Getenv("ENVO_TOKEN"),
	}
}
