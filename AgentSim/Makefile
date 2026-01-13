.PHONY: build run test clean install-deps

BINARY_NAME=agentsim
MAIN_PATH=./cmd/agentsim

build:
	@echo "Building AgentSim..."
	go build -o bin/$(BINARY_NAME) $(MAIN_PATH)

run: build
	@echo "Running AgentSim..."
	./bin/$(BINARY_NAME) --auto-start --active 50

run-many: build
	@echo "Running AgentSim with 150 active agents..."
	./bin/$(BINARY_NAME) --auto-start --active 150

run-dev: build
	@echo "Running AgentSim in dev mode..."
	./bin/$(BINARY_NAME) --log-level debug --auto-start --active 100

test:
	@echo "Running tests..."
	go test -v ./...

test-coverage:
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

clean:
	@echo "Cleaning..."
	rm -rf bin/
	rm -f coverage.out coverage.html

install-deps:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

fmt:
	@echo "Formatting code..."
	go fmt ./...

lint:
	@echo "Running linter..."
	golangci-lint run

docker-build:
	@echo "Building Docker image..."
	docker build -t agentsim:latest .

docker-run:
	@echo "Running in Docker..."
	docker run -p 8081:8081 agentsim:latest --auto-start --active 100
