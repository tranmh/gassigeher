package services

import (
	"fmt"
	"log"
	"net/http"
	"runtime"
	"time"

	"github.com/getsentry/sentry-go"
)

// SentryConfig holds Sentry configuration
type SentryConfig struct {
	DSN         string
	Environment string
	Release     string
	ServerName  string
}

// SentryService handles error tracking via Sentry
type SentryService struct {
	config  *SentryConfig
	enabled bool
}

// NewSentryService creates a new Sentry service
// Returns a no-op service if DSN is empty (disabled)
func NewSentryService(config *SentryConfig) (*SentryService, error) {
	s := &SentryService{
		config:  config,
		enabled: config.DSN != "",
	}

	if !s.enabled {
		log.Printf("Sentry: Disabled (no DSN configured)")
		return s, nil
	}

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              config.DSN,
		Environment:      config.Environment,
		Release:          config.Release,
		ServerName:       config.ServerName,
		AttachStacktrace: true,
		BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
			// You can modify or filter events here
			return event
		},
	})
	if err != nil {
		return nil, fmt.Errorf("sentry init failed: %w", err)
	}

	log.Printf("Sentry: Initialized (env=%s, release=%s)", config.Environment, config.Release)
	return s, nil
}

// CaptureException captures an error and sends it to Sentry
func (s *SentryService) CaptureException(err error) {
	if !s.enabled || err == nil {
		return
	}

	sentry.CaptureException(err)
}

// CaptureMessage captures a message and sends it to Sentry
func (s *SentryService) CaptureMessage(message string) {
	if !s.enabled {
		return
	}

	sentry.CaptureMessage(message)
}

// CaptureExceptionWithContext captures an error with additional context
func (s *SentryService) CaptureExceptionWithContext(err error, tags map[string]string, extra map[string]interface{}) {
	if !s.enabled || err == nil {
		return
	}

	sentry.WithScope(func(scope *sentry.Scope) {
		for k, v := range tags {
			scope.SetTag(k, v)
		}
		for k, v := range extra {
			scope.SetExtra(k, v)
		}
		sentry.CaptureException(err)
	})
}

// SetUser sets user context for subsequent events
func (s *SentryService) SetUser(userID int, email string, tenantID int, tenantSlug string) {
	if !s.enabled {
		return
	}

	sentry.ConfigureScope(func(scope *sentry.Scope) {
		scope.SetUser(sentry.User{
			ID:    fmt.Sprintf("%d", userID),
			Email: email,
		})
		scope.SetTag("tenant_id", fmt.Sprintf("%d", tenantID))
		scope.SetTag("tenant_slug", tenantSlug)
	})
}

// Flush waits for all events to be sent (call before shutdown)
func (s *SentryService) Flush(timeout time.Duration) {
	if !s.enabled {
		return
	}

	sentry.Flush(timeout)
}

// RecoveryMiddleware returns HTTP middleware that captures panics
func (s *SentryService) RecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				// Build error from panic
				var e error
				switch v := err.(type) {
				case error:
					e = v
				default:
					e = fmt.Errorf("panic: %v", v)
				}

				// Get stack trace
				buf := make([]byte, 4096)
				n := runtime.Stack(buf, false)
				stack := string(buf[:n])

				// Capture with context
				s.CaptureExceptionWithContext(e, map[string]string{
					"method": r.Method,
					"path":   r.URL.Path,
				}, map[string]interface{}{
					"url":        r.URL.String(),
					"user_agent": r.UserAgent(),
					"stack":      stack,
				})

				// Log locally
				log.Printf("PANIC RECOVERED: %v\n%s", err, stack)

				// Return 500 error
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// IsEnabled returns whether Sentry is enabled
func (s *SentryService) IsEnabled() bool {
	return s.enabled
}
