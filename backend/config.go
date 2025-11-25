package main

import (
	"fmt"
	"log"
	"os"
	"strings"
)

type Config struct {
	AppBaseURL     string
	BackendBaseURL string

	// Persistence / Xata (PostgreSQL)
	XataDatabaseURL string
	XataAPIKey      string
	DatabaseURL     string

	// Google OAuth
	GoogleClientID     string
	GoogleClientSecret string
	GoogleRedirectURL  string
	JwtSecret          string

	// Stripe
	StripeSecretKey      string
	StripePublishableKey string
	StripeWebhookSecret  string
	StripeDefaultPriceID string

	// Cloudflare (optional)
	CloudflareAPIToken string
}

func LoadConfig() (*Config, error) {
	c := &Config{
		AppBaseURL:     strings.TrimRight(getEnv("APP_BASE_URL", "http://localhost:18510"), "/"),
		BackendBaseURL: strings.TrimRight(getEnv("BACKEND_BASE_URL", "http://localhost:18511"), "/"),

		XataDatabaseURL: getEnv("XATA_DATABASE_URL", ""),
		XataAPIKey:      getEnv("XATA_API_KEY", ""),
		DatabaseURL:     getEnv("DATABASE_URL", ""),

		GoogleClientID:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleRedirectURL:  getEnv("GOOGLE_REDIRECT_URL", ""),
		JwtSecret:          getEnv("JWT_SECRET", ""),

		StripeSecretKey:      getEnv("STRIPE_SECRET_KEY", ""),
		StripePublishableKey: getEnv("STRIPE_PUBLISHABLE_KEY", ""),
		StripeWebhookSecret:  getEnv("STRIPE_WEBHOOK_SECRET", ""),
		StripeDefaultPriceID: getEnv("STRIPE_PRICE_ID", ""),

		CloudflareAPIToken: getEnv("CLOUDFLARE_API_TOKEN", ""),
	}

	if c.GoogleRedirectURL == "" && c.GoogleClientID != "" {
		// Default callback under backend host (Google must redirect to backend).
		c.GoogleRedirectURL = fmt.Sprintf("%s/callback/oauth/google", c.BackendBaseURL)
	}

	// Safe startup summary (no secrets).
	log.Printf(
		"config loaded: APP_BASE_URL=%s, BACKEND_BASE_URL=%s, GOOGLE_REDIRECT_URL=%s, DB_URL_set=%t, XATA_DB_URL_set=%t, GOOGLE_CLIENT_ID_set=%t, JWT_SECRET_set=%t, STRIPE_SECRET_KEY_set=%t, STRIPE_PRICE_ID_set=%t",
		c.AppBaseURL,
		c.BackendBaseURL,
		c.GoogleRedirectURL,
		c.DatabaseURL != "",
		c.XataDatabaseURL != "",
		c.GoogleClientID != "",
		c.JwtSecret != "",
		c.StripeSecretKey != "",
		c.StripeDefaultPriceID != "",
	)

	return c, nil
}

func getEnv(key string, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
