package config

import "os"

const defaultAddress = "127.0.0.1:8080"

type Config struct {
	Address string
}

func Load() Config {
	address := os.Getenv("API_ADDRESS")
	if address == "" {
		address = defaultAddress
	}

	return Config{Address: address}
}
