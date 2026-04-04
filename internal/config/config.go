package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const defaultMaxFileSize = int64(20 * 1024 * 1024)

type Config struct {
	MCPPort        int
	GatewayAddress string
	APIToken       string
	MaxFileSize    int64
}

func FromEnv() (Config, error) {
	cfg := Config{}

	portValue, err := requiredEnv("MCP_PORT")
	if err != nil {
		return Config{}, err
	}
	port, err := strconv.Atoi(strings.TrimSpace(portValue))
	if err != nil || port <= 0 || port > 65535 {
		return Config{}, fmt.Errorf("MCP_PORT must be a valid port")
	}
	cfg.MCPPort = port

	cfg.GatewayAddress, err = requiredEnv("GATEWAY_ADDRESS")
	if err != nil {
		return Config{}, err
	}

	cfg.APIToken = strings.TrimSpace(os.Getenv("AGYN_API_TOKEN"))

	maxSize := strings.TrimSpace(os.Getenv("MAX_FILE_SIZE"))
	if maxSize == "" {
		cfg.MaxFileSize = defaultMaxFileSize
	} else {
		parsed, err := strconv.ParseInt(maxSize, 10, 64)
		if err != nil {
			return Config{}, fmt.Errorf("MAX_FILE_SIZE must be an integer")
		}
		if parsed <= 0 {
			return Config{}, fmt.Errorf("MAX_FILE_SIZE must be positive")
		}
		cfg.MaxFileSize = parsed
	}

	return cfg, nil
}

func requiredEnv(name string) (string, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return "", fmt.Errorf("%s must be set", name)
	}
	return value, nil
}
