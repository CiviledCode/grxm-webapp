package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// ProfileConfig defines the constraints and settings for user profiles.
type ProfileConfig struct {
	RequireUniqueUsername bool   `json:"require_unique_username"`
	MinUsernameLength     int    `json:"min_username_length"`
	MaxUsernameLength     int    `json:"max_username_length"`
	BlacklistedChars      string `json:"blacklisted_chars"`
}

// AppConfig represents the application's configuration structure.
type AppConfig struct {
	Port                 string        `json:"port"`
	IAMHost              string        `json:"iam_host"`
	IAMAuthorityPassword string        `json:"iam_authority_password"`
	RedisHost            string        `json:"redis_host"`
	RedisPassword        string        `json:"redis_password"`
	RedisDB              int           `json:"redis_db"`
	CookieName           string        `json:"cookie_name"`
	DBProvider           string        `json:"db_provider"`
	MongoURI             string        `json:"mongo_uri"`
	MongoDB              string        `json:"mongo_db"`
	AdminRole            string        `json:"admin_role"`
	DefaultRole          string        `json:"default_role"`
	AuthedPath           string        `json:"authed_path"`
	UnauthedPath         string        `json:"unauthed_path"`
	Profile              ProfileConfig `json:"profile"`
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
	if cfg.AuthedPath == "" {
		cfg.AuthedPath = "/"
	}
	if cfg.UnauthedPath == "" {
		cfg.UnauthedPath = "/login"
	}

	if cfg.Profile.MinUsernameLength == 0 {
		cfg.Profile.MinUsernameLength = 3
	}
	if cfg.Profile.MaxUsernameLength == 0 {
		cfg.Profile.MaxUsernameLength = 20
	}

	return &cfg, nil
}
