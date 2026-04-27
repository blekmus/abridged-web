package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"abridged-web/backend/catalog"
	"abridged-web/backend/server"
)

func main() {
	archiveRoot := envOr("ARCHIVE_ROOT", "../archive")
	port := envOr("PORT", "8080")
	appEnv := strings.ToLower(envOr("APP_ENV", "development"))
	serveFromDisk := appEnv == "development"

	log.Printf("starting abridged-web (archive=%s env=%s port=%s)", archiveRoot, appEnv, port)

	library, err := catalog.Load(archiveRoot)
	if err != nil {
		fmt.Fprintln(os.Stdout, err)
		os.Exit(1)
	}

	router, err := server.NewRouter(library, server.Options{
		AppEnv:        appEnv,
		ServeFromDisk: serveFromDisk,
	})
	if err != nil {
		log.Fatalf("create router: %v", err)
	}

	log.Printf("abridged-web listening on http://localhost:%s (archive=%s env=%s)", port, archiveRoot, appEnv)
	if err := router.Run(":" + port); err != nil {
		log.Fatalf("listen: %v", err)
	}
}

func envOr(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
