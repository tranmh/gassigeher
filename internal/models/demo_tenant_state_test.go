package models

import (
	"testing"
	"time"
)

// TestDemoTenantState_Fields tests DemoTenantState struct fields
func TestDemoTenantState_Fields(t *testing.T) {
	now := time.Now()
	nextReset := now.Add(24 * time.Hour)

	state := &DemoTenantState{
		ID:            1,
		TenantID:      2,
		AdminPassword: "testpassword123",
		LastResetAt:   &now,
		NextResetAt:   &nextReset,
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	t.Run("ID field", func(t *testing.T) {
		if state.ID != 1 {
			t.Errorf("Expected ID 1, got %d", state.ID)
		}
	})

	t.Run("TenantID field", func(t *testing.T) {
		if state.TenantID != 2 {
			t.Errorf("Expected TenantID 2, got %d", state.TenantID)
		}
	})

	t.Run("AdminPassword field", func(t *testing.T) {
		if state.AdminPassword != "testpassword123" {
			t.Errorf("Expected AdminPassword 'testpassword123', got %s", state.AdminPassword)
		}
	})

	t.Run("LastResetAt field", func(t *testing.T) {
		if state.LastResetAt == nil {
			t.Error("Expected LastResetAt to be set")
		}
	})

	t.Run("NextResetAt field", func(t *testing.T) {
		if state.NextResetAt == nil {
			t.Error("Expected NextResetAt to be set")
		}
	})
}

// TestDemoTenantState_NilFields tests nil handling
func TestDemoTenantState_NilFields(t *testing.T) {
	state := &DemoTenantState{
		ID:            1,
		TenantID:      2,
		AdminPassword: "password",
	}

	t.Run("LastResetAt can be nil", func(t *testing.T) {
		if state.LastResetAt != nil {
			t.Error("Expected LastResetAt to be nil")
		}
	})

	t.Run("NextResetAt can be nil", func(t *testing.T) {
		if state.NextResetAt != nil {
			t.Error("Expected NextResetAt to be nil")
		}
	})
}

// TestDemoCredentials_Fields tests DemoCredentials struct fields
func TestDemoCredentials_Fields(t *testing.T) {
	creds := &DemoCredentials{
		AdminEmail:    "admin@demo.test",
		AdminPassword: "demopassword",
		NextResetAt:   "23.12.2025 00:00",
		LastResetAt:   "22.12.2025 00:00",
	}

	t.Run("AdminEmail field", func(t *testing.T) {
		if creds.AdminEmail != "admin@demo.test" {
			t.Errorf("Expected AdminEmail 'admin@demo.test', got %s", creds.AdminEmail)
		}
	})

	t.Run("AdminPassword field", func(t *testing.T) {
		if creds.AdminPassword != "demopassword" {
			t.Errorf("Expected AdminPassword 'demopassword', got %s", creds.AdminPassword)
		}
	})

	t.Run("NextResetAt field", func(t *testing.T) {
		if creds.NextResetAt == "" {
			t.Error("Expected NextResetAt to be set")
		}
	})

	t.Run("LastResetAt field", func(t *testing.T) {
		if creds.LastResetAt == "" {
			t.Error("Expected LastResetAt to be set")
		}
	})
}

// TestDemoUser_Fields tests DemoUser struct fields
func TestDemoUser_Fields(t *testing.T) {
	user := DemoUser{
		Name:     "Test Walker",
		Email:    "testuser@demo.test",
		Password: "demo1234",
		Level:    "green",
		LevelDE:  "Anfaenger",
	}

	t.Run("Name field", func(t *testing.T) {
		if user.Name != "Test Walker" {
			t.Errorf("Expected Name 'Test Walker', got %s", user.Name)
		}
	})

	t.Run("Email field", func(t *testing.T) {
		if user.Email != "testuser@demo.test" {
			t.Errorf("Expected Email 'testuser@demo.test', got %s", user.Email)
		}
	})

	t.Run("Password field", func(t *testing.T) {
		if user.Password != "demo1234" {
			t.Errorf("Expected Password 'demo1234', got %s", user.Password)
		}
	})

	t.Run("Level field", func(t *testing.T) {
		if user.Level != "green" {
			t.Errorf("Expected Level 'green', got %s", user.Level)
		}
	})

	t.Run("LevelDE field", func(t *testing.T) {
		if user.LevelDE != "Anfaenger" {
			t.Errorf("Expected LevelDE 'Anfaenger', got %s", user.LevelDE)
		}
	})
}

// TestDemoDog_Fields tests DemoDog struct fields
func TestDemoDog_Fields(t *testing.T) {
	dog := DemoDog{
		Name:     "Bella",
		Breed:    "Labrador Retriever",
		Category: "green",
	}

	t.Run("Name field", func(t *testing.T) {
		if dog.Name != "Bella" {
			t.Errorf("Expected Name 'Bella', got %s", dog.Name)
		}
	})

	t.Run("Breed field", func(t *testing.T) {
		if dog.Breed != "Labrador Retriever" {
			t.Errorf("Expected Breed 'Labrador Retriever', got %s", dog.Breed)
		}
	})

	t.Run("Category field", func(t *testing.T) {
		if dog.Category != "green" {
			t.Errorf("Expected Category 'green', got %s", dog.Category)
		}
	})
}

// TestDemoStatus_Fields tests DemoStatus struct fields
func TestDemoStatus_Fields(t *testing.T) {
	status := &DemoStatus{
		IsDemo:      true,
		NextResetAt: "23.12.2025 00:00",
	}

	t.Run("IsDemo field", func(t *testing.T) {
		if !status.IsDemo {
			t.Error("Expected IsDemo to be true")
		}
	})

	t.Run("NextResetAt field", func(t *testing.T) {
		if status.NextResetAt == "" {
			t.Error("Expected NextResetAt to be set")
		}
	})
}

// TestDemoStatus_NotDemo tests DemoStatus when not demo
func TestDemoStatus_NotDemo(t *testing.T) {
	status := &DemoStatus{
		IsDemo: false,
	}

	t.Run("IsDemo false", func(t *testing.T) {
		if status.IsDemo {
			t.Error("Expected IsDemo to be false")
		}
	})

	t.Run("NextResetAt empty when not demo", func(t *testing.T) {
		if status.NextResetAt != "" {
			t.Error("Expected NextResetAt to be empty when not demo")
		}
	})
}

// TestDemoTenantState_TimeComparison tests time field comparisons
func TestDemoTenantState_TimeComparison(t *testing.T) {
	now := time.Now()
	past := now.Add(-24 * time.Hour)
	future := now.Add(24 * time.Hour)

	state := &DemoTenantState{
		ID:            1,
		TenantID:      1,
		AdminPassword: "password",
		LastResetAt:   &past,
		NextResetAt:   &future,
	}

	t.Run("LastResetAt is in the past", func(t *testing.T) {
		if !state.LastResetAt.Before(now) {
			t.Error("Expected LastResetAt to be in the past")
		}
	})

	t.Run("NextResetAt is in the future", func(t *testing.T) {
		if !state.NextResetAt.After(now) {
			t.Error("Expected NextResetAt to be in the future")
		}
	})
}

// TestDemoUser_Levels tests all user levels
func TestDemoUser_Levels(t *testing.T) {
	levels := []struct {
		level   string
		levelDE string
	}{
		{"green", "Anfaenger"},
		{"orange", "Fortgeschritten"},
		{"blue", "Experte"},
	}

	for _, l := range levels {
		t.Run("Level_"+l.level, func(t *testing.T) {
			user := DemoUser{
				Name:     "Test User",
				Email:    "test@demo.test",
				Password: "demo1234",
				Level:    l.level,
				LevelDE:  l.levelDE,
			}

			if user.Level != l.level {
				t.Errorf("Expected Level %s, got %s", l.level, user.Level)
			}
			if user.LevelDE != l.levelDE {
				t.Errorf("Expected LevelDE %s, got %s", l.levelDE, user.LevelDE)
			}
		})
	}
}

// TestDemoDog_Categories tests all dog categories
func TestDemoDog_Categories(t *testing.T) {
	categories := []string{"green", "orange", "blue"}

	for _, cat := range categories {
		t.Run("Category_"+cat, func(t *testing.T) {
			dog := DemoDog{
				Name:     "Test Dog",
				Breed:    "Test Breed",
				Category: cat,
			}

			if dog.Category != cat {
				t.Errorf("Expected Category %s, got %s", cat, dog.Category)
			}
		})
	}
}
