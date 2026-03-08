package iam

import (
	"crypto/rsa"
	"fmt"
	"log"
	"net/url"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/net/websocket"
)

// Config holds the configuration required to connect to the grxm-iam service.
type Config struct {
	IAMHost           string // e.g., "localhost:8081"
	AuthorityPath     string // typically "/api/v1/authority"
	AuthorityPassword string
}

// Client represents the active IAM integration client that holds the fetched cryptographic keys.
type Client struct {
	PublicKey    *rsa.PublicKey
	PublicKeyPEM string
}

// NewClient connects to the grxm-iam Authority WebSocket, authenticates, and fetches the RSA public key.
func NewClient(cfg Config) (*Client, error) {
	if cfg.IAMHost == "" {
		return nil, fmt.Errorf("IAMHost is required")
	}
	if cfg.AuthorityPath == "" {
		cfg.AuthorityPath = "/api/v1/authority"
	}

	// Prepare the WebSocket URL with authentication query parameter
	u := url.URL{Scheme: "ws", Host: cfg.IAMHost, Path: cfg.AuthorityPath}
	q := u.Query()
	if cfg.AuthorityPassword != "" {
		q.Set("auth", cfg.AuthorityPassword)
	}
	u.RawQuery = q.Encode()

	// The Origin header is required by x/net/websocket
	origin := "http://localhost/"

	log.Printf("Connecting to IAM Authority WebSocket at %s...", u.String())
	ws, err := websocket.Dial(u.String(), "", origin)
	if err != nil {
		return nil, fmt.Errorf("failed to dial authority websocket: %w", err)
	}
	defer ws.Close()

	// 1. Send the public_key request over the WebSocket
	req := map[string]string{"action": "public_key"}
	if err := websocket.JSON.Send(ws, req); err != nil {
		return nil, fmt.Errorf("failed to send public_key request: %w", err)
	}

	// 2. Read and parse the response
	var resp struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}

	if err := websocket.JSON.Receive(ws, &resp); err != nil {
		return nil, fmt.Errorf("failed to receive public_key response: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("authority rejected request: %s", resp.Message)
	}

	// 3. Parse the retrieved PEM encoded public key
	pubKey, err := jwt.ParseRSAPublicKeyFromPEM([]byte(resp.Message))
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key PEM: %w", err)
	}

	log.Println("Successfully retrieved and parsed IAM public key via Authority WebSocket")

	return &Client{
		PublicKey:    pubKey,
		PublicKeyPEM: resp.Message,
	}, nil
}
