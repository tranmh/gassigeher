package static

import (
	"embed"
	"io/fs"
)

//go:embed frontend
var frontendFiles embed.FS

// FrontendFS returns the frontend filesystem (without the "frontend" prefix)
func FrontendFS() (fs.FS, error) {
	return fs.Sub(frontendFiles, "frontend")
}

// FrontendFile returns a specific file from the embedded frontend
func FrontendFile(path string) ([]byte, error) {
	return frontendFiles.ReadFile("frontend/" + path)
}
