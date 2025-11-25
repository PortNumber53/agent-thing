package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"gopkg.in/ini.v1"
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
	// Load optional INI config (prod default: /etc/agent-thing/config.ini).
	iniCfg := loadIniConfig()

	// Env/.env take precedence over INI when present.
	c := &Config{
		AppBaseURL:     strings.TrimRight(firstNonEmpty(getEnvOptional("APP_BASE_URL"), iniCfg.AppBaseURL, "http://localhost:18510"), "/"),
		BackendBaseURL: strings.TrimRight(firstNonEmpty(getEnvOptional("BACKEND_BASE_URL"), iniCfg.BackendBaseURL, "http://localhost:18511"), "/"),

		XataDatabaseURL: firstNonEmpty(getEnvOptional("XATA_DATABASE_URL"), iniCfg.XataDatabaseURL, ""),
		XataAPIKey:      firstNonEmpty(getEnvOptional("XATA_API_KEY"), iniCfg.XataAPIKey, ""),
		DatabaseURL:     firstNonEmpty(getEnvOptional("DATABASE_URL"), iniCfg.DatabaseURL, ""),

		GoogleClientID:     firstNonEmpty(getEnvOptional("GOOGLE_CLIENT_ID"), iniCfg.GoogleClientID, ""),
		GoogleClientSecret: firstNonEmpty(getEnvOptional("GOOGLE_CLIENT_SECRET"), iniCfg.GoogleClientSecret, ""),
		GoogleRedirectURL:  firstNonEmpty(getEnvOptional("GOOGLE_REDIRECT_URL"), iniCfg.GoogleRedirectURL, ""),
		JwtSecret:          firstNonEmpty(getEnvOptional("JWT_SECRET"), iniCfg.JwtSecret, ""),

		StripeSecretKey:      firstNonEmpty(getEnvOptional("STRIPE_SECRET_KEY"), iniCfg.StripeSecretKey, ""),
		StripePublishableKey: firstNonEmpty(getEnvOptional("STRIPE_PUBLISHABLE_KEY"), iniCfg.StripePublishableKey, ""),
		StripeWebhookSecret:  firstNonEmpty(getEnvOptional("STRIPE_WEBHOOK_SECRET"), iniCfg.StripeWebhookSecret, ""),
		StripeDefaultPriceID: firstNonEmpty(getEnvOptional("STRIPE_PRICE_ID"), iniCfg.StripeDefaultPriceID, ""),

		CloudflareAPIToken: firstNonEmpty(getEnvOptional("CLOUDFLARE_API_TOKEN"), iniCfg.CloudflareAPIToken, ""),
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

func getEnvOptional(key string) string {
	return strings.TrimSpace(os.Getenv(key))
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}

// loadIniConfig tries CONFIG_INI_PATH or /etc/agent-thing/config.ini.
// Missing/invalid file is not fatal; we just log and continue with env defaults.
func loadIniConfig() *Config {
	path := strings.TrimSpace(os.Getenv("CONFIG_INI_PATH"))
	if path == "" {
		path = "/etc/agent-thing/config.ini"
	}

	if _, err := os.Stat(path); err != nil {
		log.Printf("ini config not found at %s (ok for dev): %v", path, err)
		return &Config{}
	}

	f, err := ini.Load(path)
	if err != nil {
		log.Printf("failed to load ini config at %s (ignored): %v", path, err)
		return &Config{}
	}

	sec := f.Section("")
	c := &Config{
		AppBaseURL:           sec.Key("APP_BASE_URL").String(),
		BackendBaseURL:       sec.Key("BACKEND_BASE_URL").String(),
		XataDatabaseURL:      sec.Key("XATA_DATABASE_URL").String(),
		XataAPIKey:           sec.Key("XATA_API_KEY").String(),
		DatabaseURL:          sec.Key("DATABASE_URL").String(),
		GoogleClientID:       sec.Key("GOOGLE_CLIENT_ID").String(),
		GoogleClientSecret:   sec.Key("GOOGLE_CLIENT_SECRET").String(),
		GoogleRedirectURL:    sec.Key("GOOGLE_REDIRECT_URL").String(),
		JwtSecret:            sec.Key("JWT_SECRET").String(),
		StripeSecretKey:      sec.Key("STRIPE_SECRET_KEY").String(),
		StripePublishableKey: sec.Key("STRIPE_PUBLISHABLE_KEY").String(),
		StripeWebhookSecret:  sec.Key("STRIPE_WEBHOOK_SECRET").String(),
		StripeDefaultPriceID: sec.Key("STRIPE_PRICE_ID").String(),
		CloudflareAPIToken:   sec.Key("CLOUDFLARE_API_TOKEN").String(),
	}
	log.Printf("loaded ini config from %s", path)
	return c
}
