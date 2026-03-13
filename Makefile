.PHONY: build run test clean lint

BINARY=clawtrade
BUILD_DIR=bin

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/clawtrade

run: build
	./$(BUILD_DIR)/$(BINARY) serve

test:
	go test ./... -v -race

clean:
	rm -rf $(BUILD_DIR)

lint:
	golangci-lint run ./...
