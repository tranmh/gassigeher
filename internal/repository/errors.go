package repository

import "errors"

// Common repository errors for tenant isolation and resource not found scenarios
var (
	// ErrNotFound is returned when the requested resource doesn't exist
	ErrNotFound = errors.New("resource not found")

	// ErrTenantMismatch is returned when a resource exists but belongs to a different tenant
	// This is a security-sensitive error that indicates a potential cross-tenant access attempt
	ErrTenantMismatch = errors.New("resource belongs to a different tenant")
)
