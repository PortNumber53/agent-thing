package config

import (
	"fmt"
	"log"

	"gopkg.in/ini.v1"
)

// Config holds the application configuration.
type Config struct {
	DBHost         string
	DBPort         string
	DBName         string
	DBUser         string
	DBPassword     string
	DBSslMode      string
	GeminiAPIKey   string
	ChrootDir      string
	GeminiModel    string
}

// LoadConfig reads the configuration from the given path.
func LoadConfig(path string) (*Config, error) {
	cfg, err := ini.Load(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := &Config{
		DBHost:       cfg.Section("default").Key("DB_HOST").String(),
		DBPort:       cfg.Section("default").Key("DB_PORT").String(),
		DBName:       cfg.Section("default").Key("DB_NAME").String(),
		DBUser:       cfg.Section("default").Key("DB_USER").String(),
		DBPassword:   cfg.Section("default").Key("DB_PASSWORD").String(),
		DBSslMode:    cfg.Section("default").Key("DB_SSLMODE").String(),
		GeminiAPIKey: cfg.Section("default").Key("GEMINI_API_KEY").String(),
		ChrootDir:    cfg.Section("default").Key("CHROOT_DIR").String(),
		GeminiModel:  cfg.Section("default").Key("GEMINI_MODEL").String(),
	}

	if config.GeminiAPIKey == "" {
		log.Println("Warning: GEMINI_API_KEY is not set in the config file.")
	}

	return config, nil
}
