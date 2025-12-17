package static

import (
	"embed"
	"io/fs"
)

//go:embed frontend
var frontendFiles embed.FS

//go:embed landing
var landingFiles embed.FS

// FrontendFS returns the frontend filesystem (without the "frontend" prefix)
func FrontendFS() (fs.FS, error) {
	return fs.Sub(frontendFiles, "frontend")
}

// FrontendFile returns a specific file from the embedded frontend
func FrontendFile(path string) ([]byte, error) {
	return frontendFiles.ReadFile("frontend/" + path)
}

// LandingFS returns the landing page filesystem (without the "landing" prefix)
func LandingFS() (fs.FS, error) {
	return fs.Sub(landingFiles, "landing")
}

// LandingFile returns a specific file from the embedded landing pages
func LandingFile(path string) ([]byte, error) {
	return landingFiles.ReadFile("landing/" + path)
}
