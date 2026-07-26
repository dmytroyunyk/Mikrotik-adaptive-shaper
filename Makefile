.PHONY: build test lint run docker-build docker-up docker-down
build:
	go build -o bin/adaptive-shaper .
test:
	go test ./...
lint:
	go build ./... && go vet ./...
run:
	go run .
docker-build:
	docker compose build
docker-up:
	docker compose up -d
docker-down:
	docker compose down