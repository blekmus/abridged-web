.PHONY: build build-frontend build-backend

build: build-frontend build-backend

build-frontend:
	bun run build

build-backend:
	APP_ENV=production go build -o abridged ./main.go