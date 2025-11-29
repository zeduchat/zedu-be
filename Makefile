include app.env
export

MIGRATIONS_PATH = db/migrations

migrate-up:
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" up

migrate-down:
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" down

migrate-force:
	@echo "⚙️  Forcing migration version to $(version)"
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" force $(version)

migrate-version:
	migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" version

fix-dirty:
	@echo "Checking for dirty migration state..."
	@MIGRATION_OUTPUT=$$(migrate -path $(MIGRATIONS_PATH) -database "$(DB_URL)" version 2>&1 || true); \
	echo "$$MIGRATION_OUTPUT"; \
	if echo "$$MIGRATION_OUTPUT" | grep -q "dirty"; then \
		VERSION=$$(echo "$$MIGRATION_OUTPUT" | grep -oE '^[0-9]+'); \
		PREV_VERSION=$$((VERSION - 1)); \
		echo "Dirty state detected. Forcing to version $$PREV_VERSION"; \
		make migrate-force version=$$PREV_VERSION; \
		make migrate-down; \
	else \
		echo "Migration state is clean. No action needed."; \
	fi

migrate-safe-up:
	@echo "Running safe migration..."
	@make fix-dirty
	@make migrate-up

start-dev:
	@echo "Running docker compose up..."
	docker compose -f docker-compose.dev.yml up --build

dev-clean:
	@echo "Cleaning docker compose..."
	docker compose -f docker-compose.dev.yml down -v