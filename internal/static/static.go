package static

import (
	"embed"
	"io/fs"
)

//go:embed frontend
var frontendFiles embed.FS

//go:embed landing
var landingFiles embed.FS

//go:embed central
var centralFiles embed.FS

//go:embed shared
var sharedFiles embed.FS

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

// CentralFS returns the central admin filesystem (without the "central" prefix)
func CentralFS() (fs.FS, error) {
	return fs.Sub(centralFiles, "central")
}

// CentralFile returns a specific file from the embedded central admin pages
func CentralFile(path string) ([]byte, error) {
	return centralFiles.ReadFile("central/" + path)
}

// SharedFS returns the shared filesystem (without the "shared" prefix)
func SharedFS() (fs.FS, error) {
	return fs.Sub(sharedFiles, "shared")
}

// SharedFile returns a specific file from the embedded shared files
func SharedFile(path string) ([]byte, error) {
	return sharedFiles.ReadFile("shared/" + path)
}
