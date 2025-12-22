package services

import (
	"runtime"
	"testing"
	"time"
)

// TestBruteForceService_GoroutineLeak tests that BruteForceService can be stopped
// and doesn't leak goroutines when multiple instances are created
func TestBruteForceService_GoroutineLeak(t *testing.T) {
	// Get baseline goroutine count
	runtime.GC()
	baselineGoroutines := runtime.NumGoroutine()

	// Create multiple instances
	services := make([]*BruteForceService, 5)
	for i := 0; i < 5; i++ {
		services[i] = NewBruteForceService()
	}

	// Give goroutines time to start
	time.Sleep(10 * time.Millisecond)

	// Verify goroutines were created
	afterCreateGoroutines := runtime.NumGoroutine()
	if afterCreateGoroutines <= baselineGoroutines {
		t.Logf("Warning: Expected more goroutines after creating services (baseline: %d, after: %d)",
			baselineGoroutines, afterCreateGoroutines)
	}

	// Stop all services
	for _, svc := range services {
		svc.Stop()
	}

	// Give goroutines time to stop
	time.Sleep(50 * time.Millisecond)
	runtime.GC()

	// Check goroutine count returned to baseline
	afterStopGoroutines := runtime.NumGoroutine()

	// Allow some tolerance (1-2 goroutines for runtime overhead)
	if afterStopGoroutines > baselineGoroutines+2 {
		t.Errorf("Goroutine leak detected: baseline=%d, afterCreate=%d, afterStop=%d",
			baselineGoroutines, afterCreateGoroutines, afterStopGoroutines)
	}
}

// TestBruteForceService_BasicFunctionality tests basic brute force protection
func TestBruteForceService_BasicFunctionality(t *testing.T) {
	svc := NewBruteForceService()
	defer svc.Stop()

	key := "test@example.com:192.168.1.1"

	// Initially not locked
	locked, _ := svc.IsLocked(key)
	if locked {
		t.Error("Should not be locked initially")
	}

	// Record failures
	for i := 0; i < 3; i++ {
		svc.RecordFailure(key)
	}

	// Should now be locked after 3 failures
	locked, remaining := svc.IsLocked(key)
	if !locked {
		t.Error("Should be locked after 3 failures")
	}
	if remaining <= 0 {
		t.Error("Should have remaining lockout time")
	}

	// Clear failures
	svc.ClearFailures(key)

	// Should be unlocked after clearing
	locked, _ = svc.IsLocked(key)
	if locked {
		t.Error("Should not be locked after clearing failures")
	}
}

// TestBruteForceService_FailureCount tests failure counting
func TestBruteForceService_FailureCount(t *testing.T) {
	svc := NewBruteForceService()
	defer svc.Stop()

	key := "user@test.com:10.0.0.1"

	// Initially 0
	if count := svc.GetFailureCount(key); count != 0 {
		t.Errorf("Expected 0 failures, got %d", count)
	}

	// Record 2 failures
	svc.RecordFailure(key)
	svc.RecordFailure(key)

	if count := svc.GetFailureCount(key); count != 2 {
		t.Errorf("Expected 2 failures, got %d", count)
	}
}
