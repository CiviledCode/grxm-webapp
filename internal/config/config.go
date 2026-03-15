package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// AppConfig represents the application's configuration structure.
type AppConfig struct {
	Port                 string `json:"port"`
	IAMHost              string `json:"iam_host"`
	IAMAuthorityPassword string `json:"iam_authority_password"`
	RedisHost            string `json:"redis_host"`
	RedisPassword        string `json:"redis_password"`
	RedisDB              int    `json:"redis_db"`
	CookieName           string `json:"cookie_name"`
	DBProvider           string `json:"db_provider"`
	MongoURI             string `json:"mongo_uri"`
	MongoDB              string `json:"mongo_db"`
}

// LoadConfig attempts to locate and parse the config.json file based on priority:
// 1. API_CONFIG_LOCATION environment variable
// 2. config.json in the current directory
func LoadConfig() (*AppConfig, error) {
	configPaths := []string{}

	// Priority 1: API_CONFIG_LOCATION
	if envPath := os.Getenv("API_CONFIG_LOCATION"); envPath != "" {
		configPaths = append(configPaths, envPath)
	}

	// Priority 2: Current directory fallback
	configPaths = append(configPaths, "config.json")

	var fileData []byte
	var loadedPath string
	var err error

	// Attempt to read the file from the prioritized paths
	for _, path := range configPaths {
		fileData, err = os.ReadFile(path)
		if err == nil {
			loadedPath = path
			break
		}
	}

	if fileData == nil {
		return nil, fmt.Errorf("could not find or read config file in any of the checked locations: %v", configPaths)
	}

	fmt.Printf("Loaded configuration from: %s\n", loadedPath)

	var cfg AppConfig
	if err := json.Unmarshal(fileData, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config JSON at %s: %w", loadedPath, err)
	}

	// Set safe defaults if fields are missing in the JSON
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if cfg.IAMHost == "" {
		cfg.IAMHost = "localhost:8081"
	}
	if cfg.RedisHost == "" {
		cfg.RedisHost = "localhost:6379"
	}
	if cfg.CookieName == "" {
		cfg.CookieName = "grxm-token"
	}
	if cfg.DBProvider == "" {
		cfg.DBProvider = "mongo"
	}
	if cfg.MongoURI == "" {
		cfg.MongoURI = "mongodb://localhost:27017"
	}
	if cfg.MongoDB == "" {
		cfg.MongoDB = "grxm_webapp"
	}

	return &cfg, nil
}
