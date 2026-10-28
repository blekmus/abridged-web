package server

import "io/fs"

// The frontend build renders this shell from the same App the browser hydrates.
// Only the homepage is prerendered; other routes mount into an empty root.
func appShellForPath(assets fs.FS, requestPath string) (string, error) {
	if requestPath != "/" && requestPath != "" {
		return "", nil
	}

	data, err := fs.ReadFile(assets, "app-shell.html")
	return string(data), err
}
