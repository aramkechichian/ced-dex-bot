.PHONY: help run pretty simulate simulate-notify test build vet docker-up docker-down docker-simulate docker-simulate-notify clean

help:
	@echo "Targets:"
	@echo "  make run              Run bot (live mainnet)"
	@echo "  make pretty           Live run with --pretty (3 blocks)"
	@echo "  make simulate         Offline demo + pretty table"
	@echo "  make simulate-notify  Offline demo + action webhook (needs ACTION_WEBHOOK_URL)"
	@echo "  make test             Run unit tests"
	@echo "  make build            Build bin/bot"
	@echo "  make docker-up        docker compose up --build"
	@echo "  make docker-simulate  Simulate inside Docker (rebuilds image)"
	@echo "  make docker-simulate-notify  Simulate + action webhook in Docker"

run:
	go run ./cmd/bot

pretty:
	go run ./cmd/bot --pretty -blocks 3

simulate:
	go run ./cmd/bot -simulate --pretty

simulate-notify:
	go run ./cmd/bot -simulate --pretty -notify

test:
	go test ./...

vet:
	go vet ./...

build:
	go build -o bin/bot ./cmd/bot

docker-up:
	docker compose up --build

docker-down:
	docker compose down

docker-simulate:
	docker compose run --rm --build bot -config /app/configs/config.yaml -simulate --pretty

docker-simulate-notify:
	docker compose run --rm --build bot -config /app/configs/config.yaml -simulate --pretty -notify

clean:
	rm -f bin/bot
