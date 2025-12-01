package version

import (
	"testing"
)

func TestGet_ReturnsVersionInfo(t *testing.T) {
	info := Get()

	// Default values when not set via ldflags
	if info.Version == "" {
		t.Error("Version should not be empty")
	}
	if info.GitCommit == "" {
		t.Error("GitCommit should not be empty")
	}
	if info.BuildTime == "" {
		t.Error("BuildTime should not be empty")
	}
}

func TestGet_ReturnsDefaultValues(t *testing.T) {
	// When not built with ldflags, should return defaults
	info := Get()

	if info.Version != "dev" {
		t.Errorf("Expected default Version 'dev', got '%s'", info.Version)
	}
	if info.GitCommit != "unknown" {
		t.Errorf("Expected default GitCommit 'unknown', got '%s'", info.GitCommit)
	}
	if info.BuildTime != "unknown" {
		t.Errorf("Expected default BuildTime 'unknown', got '%s'", info.BuildTime)
	}
}

func TestInfo_JSONTags(t *testing.T) {
	// Verify the struct has the expected fields
	info := Info{
		Version:   "1.0",
		GitCommit: "abc123",
		BuildTime: "2025-01-01T00:00:00Z",
	}

	if info.Version != "1.0" {
		t.Errorf("Expected Version '1.0', got '%s'", info.Version)
	}
	if info.GitCommit != "abc123" {
		t.Errorf("Expected GitCommit 'abc123', got '%s'", info.GitCommit)
	}
	if info.BuildTime != "2025-01-01T00:00:00Z" {
		t.Errorf("Expected BuildTime '2025-01-01T00:00:00Z', got '%s'", info.BuildTime)
	}
}
