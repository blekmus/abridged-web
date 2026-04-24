package main

import (
	"log"
	"os"
	"strings"

	"abridged-web/backend/catalog"
	"abridged-web/backend/server"
)

func main() {
	archiveRoot := envOr("ARCHIVE_ROOT", "../archive")
	port := envOr("PORT", "8080")
	appEnv := strings.ToLower(envOr("APP_ENV", "production"))
	serveFromDisk := appEnv == "development"

	library, err := catalog.Load(archiveRoot)
	if err != nil {
		log.Fatalf("load archive: %v", err)
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
