package profile

import (
	"context"
	"errors"
	"strings"

	"github.com/civiledcode/grxm-webapp/internal/config"
	"github.com/civiledcode/grxm-webapp/internal/db"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

var (
	ErrProfileNotFound = errors.New("profile not found")
	ErrUsernameTaken   = errors.New("username is already taken")
	ErrInvalidUsername = errors.New("invalid username length or contains blacklisted characters")
)

// Profile represents the user's application profile
type Profile struct {
	UUID     string `bson:"_id" json:"uuid"`
	Username string `bson:"username" json:"username"`
}

// Init configures the MongoDB collection and unique indexes based on config
func Init(ctx context.Context, cfg *config.AppConfig) error {
	col := db.MongoDB.Collection("profiles")

	if cfg.Profile.RequireUniqueUsername {
		_, err := col.Indexes().CreateOne(ctx, mongo.IndexModel{
			Keys:    bson.D{{Key: "username", Value: 1}},
			Options: options.Index().SetUnique(true),
		})
		if err != nil {
			return err
		}
	}
	return nil
}

// Get retrieves a Profile by the IAM UUID
func Get(ctx context.Context, uuid string) (*Profile, error) {
	var p Profile
	err := db.MongoDB.Collection("profiles").FindOne(ctx, bson.M{"_id": uuid}).Decode(&p)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrProfileNotFound
		}
		return nil, err
	}
	return &p, nil
}

// Create validates config constraints and creates a new profile
func Create(ctx context.Context, cfg *config.AppConfig, uuid, username string) (*Profile, error) {
	username = strings.TrimSpace(username)

	// Validate length
	if len(username) < cfg.Profile.MinUsernameLength || len(username) > cfg.Profile.MaxUsernameLength {
		return nil, ErrInvalidUsername
	}

	// Validate blacklist
	if cfg.Profile.BlacklistedChars != "" && strings.ContainsAny(username, cfg.Profile.BlacklistedChars) {
		return nil, ErrInvalidUsername
	}

	p := &Profile{
		UUID:     uuid,
		Username: username,
	}

	_, err := db.MongoDB.Collection("profiles").InsertOne(ctx, p)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, ErrUsernameTaken
		}
		return nil, err
	}

	return p, nil
}
