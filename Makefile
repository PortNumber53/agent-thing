# Makefile

# Load database configuration from .ini file
CONFIG_FILE=$(HOME)/.config/agent-thing/config.ini
DB_USER := $(shell awk -F'=' '/^DB_USER/ {gsub(/[ \t]/, "", $$2); print $$2}' $(CONFIG_FILE))
DB_PASSWORD := $(shell awk -F'=' '/^DB_PASSWORD/ {gsub(/[ \t]/, "", $$2); print $$2}' $(CONFIG_FILE))
DB_HOST := $(shell awk -F'=' '/^DB_HOST/ {gsub(/[ \t]/, "", $$2); print $$2}' $(CONFIG_FILE))
DB_PORT := $(shell awk -F'=' '/^DB_PORT/ {gsub(/[ \t]/, "", $$2); print $$2}' $(CONFIG_FILE))
DB_NAME := $(shell awk -F'=' '/^DB_NAME/ {gsub(/[ \t]/, "", $$2); print $$2}' $(CONFIG_FILE))
DB_SSLMODE := $(shell awk -F'=' '/^DB_SSLMODE/ {gsub(/[ \t]/, "", $$2); print $$2}' $(CONFIG_FILE))

DB_URL=postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)

.PHONY: migrateup migratedown force new_migration

migrateup:
	@echo "Running migrations up on $(DB_HOST)..."
	migrate -path db/migrations -database "$(DB_URL)" -verbose up

migratedown:
	@echo "Running migrations down on $(DB_HOST)..."
	migrate -path db/migrations -database "$(DB_URL)" -verbose down

force:
	@echo "Forcing migration version on $(DB_HOST)..."
	migrate -path db/migrations -database "$(DB_URL)" force

.PHONY: new_migration
new_migration:
	@read -p "Enter migration name: " name; \
	migrate create -ext sql -dir db/migrations -seq $$name
