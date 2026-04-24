package frontendassets

import "embed"

// DistFS contains the built frontend bundle for production serving.
//
//go:embed dist/*
var DistFS embed.FS
