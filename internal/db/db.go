package db

import (
	"context"
	"fmt"

	"github.com/civiledcode/grxm-webapp/internal/config"
)

// Init establishes a connection to the configured database provider.
func Init(cfg *config.AppConfig) error {
	switch cfg.DBProvider {
	case "mongo":
		return initMongo(cfg)
	default:
		return fmt.Errorf("unsupported database provider: %s", cfg.DBProvider)
	}
}

// Disconnect gracefully closes the active database connection.
func Disconnect(ctx context.Context, cfg *config.AppConfig) error {
	switch cfg.DBProvider {
	case "mongo":
		return disconnectMongo(ctx)
	default:
		return nil
	}
}
