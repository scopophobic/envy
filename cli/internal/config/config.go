package config

import "os"

type Config struct {
	APIBaseURL string
}

func Load() Config {
	base := os.Getenv("ENVO_API_URL")
	if base == "" {
		base = "http://localhost:8080"
	}

	return Config{
		APIBaseURL: base,
	}
}

