.PHONY: test test-integration test-all setup-teardown test-deps test-env

TEST_ENV_FILE := ./.env

test-deps:
	go get github.com/stretchr/testify/assert
	go get github.com/stretchr/testify/suite

test-env:
	@if [ ! -f $(TEST_ENV_FILE) ]; then \
		echo "📝 Creating test environment file..."; \
		cp /.env.example $(TEST_ENV_FILE); \
		echo "✅ Test environment file created: $(TEST_ENV_FILE)"; \
	else \
		echo "✅ Test environment file exists: $(TEST_ENV_FILE)"; \
	fi

test-all: test-env test-integration

test-integration: test-env
	@echo "🚀 Running Integration Tests..."
	@export $$(cat $(TEST_ENV_FILE) | xargs) && \
	cd integration && go test -v -timeout=5m


test-up: test-env
	@echo "🐳 Starting test environment..."
	@export $$(cat $(TEST_ENV_FILE) | xargs) && \
	docker-compose -f docker-compose.test.yml up --build -d
	@echo "✅ Test environment ready. Run tests with: make test-integration"

test-down:
	@echo "🧹 Stopping test environment..."
	@docker-compose -f docker-compose.test.yml down
	@echo "✅ Test environment stopped"

setup-teardown: test-env test-up
	@sleep 10
	@echo "⏳ Waiting for services to be ready..."
	@until curl -s http://localhost:$${PORT:-8080}/stats/system > /dev/null; do \
		sleep 2; \
	done
	@$(MAKE) test-integration || true
	@$(MAKE) test-down

test-coverage: test-env test-up
	@sleep 10
	@echo "📊 Running tests with coverage..."
	@export $$(cat $(TEST_ENV_FILE) | xargs) && \
	cd integration && go test -v -coverprofile=coverage.out -timeout=5m
	@go tool cover -html=integration/coverage.out -o coverage.html
	@echo "📈 Coverage report generated: coverage.html"
	@$(MAKE) test-down

test-quick: test-env test-up
	@sleep 10
	@echo "⚡ Running Quick Tests..."
	@export $$(cat $(TEST_ENV_FILE) | xargs) && \
	cd integration && go test -v -run="TestTeamWorkflow|TestPRWorkflow" -timeout=2m
	@$(MAKE) test-down

test-ci: test-deps
	@echo "🏗️ Running CI Test Suite..."
	@docker-compose -f docker-compose.test.yml up --build -d
	@sleep 15
	@cd integration && go test -v -timeout=5m
	@docker-compose -f docker-compose.test.yml down

test-clean:
	@echo "🧹 Cleaning test resources..."
	@docker-compose -f docker-compose.test.yml down -v
	@rm -f integration/coverage.out coverage.html
	@echo "✅ Test resources cleaned"