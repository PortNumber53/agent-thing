package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

// loadDotEnv loads environment variables from .env files if present.
// Order (later files override earlier):
// 1. repo root .env
// 2. backend/.env
// This is optional; missing files are ignored.
func loadDotEnv() {
	// Resolve candidate paths relative to the project root if possible.
	rootDir := ""
	if wd, err := os.Getwd(); err == nil {
		if filepath.Base(wd) == "backend" {
			rootDir = filepath.Dir(wd)
		} else {
			rootDir = wd
		}
	}

	paths := []string{
		filepath.Join(rootDir, ".env"),
		filepath.Join(rootDir, "backend", ".env"),
		// Back-compat for odd working dirs.
		".env",
		"backend/.env",
		"../.env",
	}
	loadedAny := false
	for _, p := range paths {
		if p == "" {
			continue
		}

		if _, err := os.Stat(p); err == nil {
			if err := godotenv.Overload(p); err != nil {
				log.Printf("warning: failed to load %s: %v", p, err)
			} else {
				log.Printf("loaded env from %s", p)
				loadedAny = true
			}
		}
	}
	if !loadedAny {
		log.Printf("no .env file found (looked for %v); using process environment only", paths)
	}
}
