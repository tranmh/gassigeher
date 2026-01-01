package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/tranmh/gassigeher/internal/database"
)

// TestHealthHandler_Health tests the health check endpoint
func TestHealthHandler_Health(t *testing.T) {
	// Create in-memory database for testing
	rawDB, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to create test database: %v", err)
	}
	defer rawDB.Close()

	// Wrap in database.DB for auto-rebinding
	sqlxDB := sqlx.NewDb(rawDB, "sqlite3")
	db := database.WrapSqlxDB(sqlxDB, database.NewSQLiteDialect())

	handler := NewHealthHandler(db)

	t.Run("returns ok status", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/health", nil)
		rec := httptest.NewRecorder()

		handler.Health(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", rec.Code)
		}

		var response map[string]string
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to parse response: %v", err)
		}

		if response["status"] != "ok" {
			t.Errorf("Expected status 'ok', got '%s'", response["status"])
		}
	})

	t.Run("works with any HTTP method", func(t *testing.T) {
		methods := []string{"GET", "POST", "HEAD"}
		for _, method := range methods {
			req := httptest.NewRequest(method, "/api/health", nil)
			rec := httptest.NewRecorder()

			handler.Health(rec, req)

			if rec.Code != http.StatusOK {
				t.Errorf("Expected status 200 for %s, got %d", method, rec.Code)
			}
		}
	})
}
