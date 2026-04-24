# Abridged Web

## Development

Run the frontend dev server and backend seperately in two different terminals:

```sh
bun run dev
```
```sh
go run ./main.go
```

## Build

Build manually:

```sh
bun run build
APP_ENV=production go build -o abridged ./main.go
```

Or use the Makefile:

```sh
make build
```

## Runtime

Run the application using the following command. Environment variables are read at runtime from the process environment.

Environment variables:

- `PORT`: server port, defaults to `8080`
- `ARCHIVE_ROOT`: archive directory, defaults to `../archive`

Example:

```sh
PORT=3000 ARCHIVE_ROOT=/path/to/archive ./abridged
```
