package services

import (
	"sync"
	"time"
)

// BruteForceService tracks failed login attempts and implements lockout
type BruteForceService struct {
	failures    map[string]*FailureRecord
	mu          sync.RWMutex
	maxAttempts int           // Number of failures before lockout
	lockoutBase time.Duration // Base lockout duration
	maxLockout  time.Duration // Maximum lockout duration
}

// FailureRecord tracks login failures for a specific key (email:ip)
type FailureRecord struct {
	Count       int
	LastFailed  time.Time
	LockedUntil time.Time
}

// NewBruteForceService creates a new brute force protection service
func NewBruteForceService() *BruteForceService {
	bfs := &BruteForceService{
		failures:    make(map[string]*FailureRecord),
		maxAttempts: 3,
		lockoutBase: 30 * time.Second,
		maxLockout:  30 * time.Minute,
	}

	// Start cleanup goroutine
	go bfs.cleanupStaleEntries()

	return bfs
}

// RecordFailure records a failed login attempt and returns lockout duration (0 if not locked)
func (s *BruteForceService) RecordFailure(key string) time.Duration {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, exists := s.failures[key]
	if !exists {
		record = &FailureRecord{}
		s.failures[key] = record
	}

	// Reset count if last failure was more than 1 hour ago
	if time.Since(record.LastFailed) > time.Hour {
		record.Count = 0
	}

	record.Count++
	record.LastFailed = time.Now()

	if record.Count >= s.maxAttempts {
		// Exponential backoff: 30s, 60s, 120s, 240s... max 30min
		exponent := record.Count - s.maxAttempts
		if exponent > 10 {
			exponent = 10 // Cap to prevent overflow
		}
		multiplier := 1 << exponent
		delay := s.lockoutBase * time.Duration(multiplier)
		if delay > s.maxLockout {
			delay = s.maxLockout
		}
		record.LockedUntil = time.Now().Add(delay)
		return delay
	}

	return 0
}

// IsLocked checks if an account is currently locked out
func (s *BruteForceService) IsLocked(key string) (bool, time.Duration) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	record, exists := s.failures[key]
	if !exists {
		return false, 0
	}

	if time.Now().Before(record.LockedUntil) {
		return true, time.Until(record.LockedUntil)
	}

	return false, 0
}

// ClearFailures clears all failure records for a key (on successful login)
func (s *BruteForceService) ClearFailures(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.failures, key)
}

// GetFailureCount returns the current failure count for a key
func (s *BruteForceService) GetFailureCount(key string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	record, exists := s.failures[key]
	if !exists {
		return 0
	}
	return record.Count
}

// cleanupStaleEntries removes old failure records that haven't been updated in 2 hours
func (s *BruteForceService) cleanupStaleEntries() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		cutoff := time.Now().Add(-2 * time.Hour)
		for key, record := range s.failures {
			if record.LastFailed.Before(cutoff) && time.Now().After(record.LockedUntil) {
				delete(s.failures, key)
			}
		}
		s.mu.Unlock()
	}
}
