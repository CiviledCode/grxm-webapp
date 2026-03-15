package main

import (
	"log"
	"net/http"

	"github.com/civiledcode/grxm-webapp/internal/api"
	"github.com/civiledcode/grxm-webapp/internal/config"
	"github.com/civiledcode/grxm-webapp/internal/db"
	"github.com/civiledcode/grxm-webapp/internal/iam"
)

func main() {
	// 1. Load the configuration from JSON based on defined priority
	appConfig, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Fatal: Configuration error: %v", err)
	}

	if appConfig.IAMAuthorityPassword == "" {
		log.Println("WARNING: iam_authority_password is not set in the configuration.")
		log.Println("The backend may fail to connect to the IAM Authority WebSocket if it requires authentication.")
	}

	// 2. Initialize MongoDB connection
	if err := db.Init(appConfig); err != nil {
		log.Fatalf("Critical Error: Failed to initialize MongoDB connection: %v", err)
	}

	iamCfg := iam.Config{
		IAMHost:           appConfig.IAMHost,
		AuthorityPath:     "/iam/api/v1/authority",
		AuthorityPassword: appConfig.IAMAuthorityPassword,
		RedisHost:         appConfig.RedisHost,
		RedisPassword:     appConfig.RedisPassword,
		RedisDB:           appConfig.RedisDB,
		CookieName:        appConfig.CookieName,
	}

	// 2. Initialize the IAM Client
	// This immediately dials the IAM Authority WebSocket and requests the public key.
	// If the IAM service is unavailable or credentials fail, the app stops booting here.
	iamClient, err := iam.NewClient(iamCfg)
	if err != nil {
		log.Fatalf("Critical Error: Failed to initialize IAM Client: %v", err)
	}

	// 3. Configure API Routing
	mux := http.NewServeMux()

	// Public health check endpoint
	mux.HandleFunc("/health", api.HealthHandler)

	// Protected API route - Wraps HelloHandler in the JWT validation middleware
	mux.HandleFunc("/api/hello", iamClient.AuthRequired(api.HelloHandler(iamClient.PublicKeyPEM)))

	// Serve the frontend template files (HTML, CSS, JS)
	fileServer := http.FileServer(http.Dir("./static"))
	mux.Handle("/", fileServer)

	// 4. Start Server
	log.Printf("Starting Template Webapp server on port %s", appConfig.Port)
	if err := http.ListenAndServe(":"+appConfig.Port, mux); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}
