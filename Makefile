.PHONY: build build-frontend build-backend check lint format dev-frontend dev-backend

check:
	bun run check

lint:
	bun run lint

format:
	bun run format

build: build-frontend build-backend

build-frontend:
	bun run build

build-backend:
	go build -o abridged ./main.go

dev-frontend:
	bun run dev

dev-backend:
	APP_ENV=development go run ./main.go
