package repository

import (
	"testing"

	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/testutil"
)

// TestColorRequestRepository_Create tests color request creation
func TestColorRequestRepository_Create(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewColorRequestRepository(db)

	t.Run("successful creation", func(t *testing.T) {
		userID := testutil.SeedTestUser(t, db, "user1@test.com", "Test User", "green")
		colorID := testutil.SeedTestColorCategory(t, db, "req-color", "#123456", 10)

		req := &models.ColorRequest{
			UserID:  userID,
			ColorID: colorID,
			Status:  "pending",
		}

		err := repo.Create(req)
		if err != nil {
			t.Fatalf("Create() failed: %v", err)
		}

		if req.ID == 0 {
			t.Error("ColorRequest ID should be set after creation")
		}
	})
}

// TestColorRequestRepository_FindByID tests finding request by ID
func TestColorRequestRepository_FindByID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewColorRequestRepository(db)

	t.Run("request exists", func(t *testing.T) {
		userID := testutil.SeedTestUser(t, db, "user2@test.com", "Test User 2", "green")
		colorID := testutil.SeedTestColorCategory(t, db, "find-color", "#aabbcc", 20)
		reqID := testutil.SeedTestColorRequest(t, db, userID, colorID, "pending")

		request, err := repo.FindByID(reqID)
		if err != nil {
			t.Fatalf("FindByID() failed: %v", err)
		}

		if request.ID != reqID {
			t.Errorf("Expected ID %d, got %d", reqID, request.ID)
		}

		if request.ColorID != colorID {
			t.Errorf("Expected ColorID %d, got %d", colorID, request.ColorID)
		}

		if request.Status != "pending" {
			t.Errorf("Expected status 'pending', got %s", request.Status)
		}
	})

	t.Run("request not found", func(t *testing.T) {
		request, _ := repo.FindByID(99999)
		if request != nil {
			t.Error("Expected nil for non-existent ID")
		}
	})
}

// TestColorRequestRepository_FindByUserID tests finding user's requests
func TestColorRequestRepository_FindByUserID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewColorRequestRepository(db)

	user1ID := testutil.SeedTestUser(t, db, "user3@test.com", "User 3", "green")
	user2ID := testutil.SeedTestUser(t, db, "user4@test.com", "User 4", "green")
	color1ID := testutil.SeedTestColorCategory(t, db, "user-color-1", "#111111", 30)
	color2ID := testutil.SeedTestColorCategory(t, db, "user-color-2", "#222222", 40)

	// Create requests for user1
	testutil.SeedTestColorRequest(t, db, user1ID, color1ID, "pending")
	testutil.SeedTestColorRequest(t, db, user1ID, color2ID, "approved")

	// Create request for user2
	testutil.SeedTestColorRequest(t, db, user2ID, color1ID, "pending")

	t.Run("user has multiple requests", func(t *testing.T) {
		requests, err := repo.FindByUserID(user1ID)
		if err != nil {
			t.Fatalf("FindByUserID() failed: %v", err)
		}

		if len(requests) != 2 {
			t.Errorf("Expected 2 requests for user1, got %d", len(requests))
		}
	})

	t.Run("user has no requests", func(t *testing.T) {
		user3ID := testutil.SeedTestUser(t, db, "user5@test.com", "User 5", "green")

		requests, err := repo.FindByUserID(user3ID)
		if err != nil {
			t.Fatalf("FindByUserID() failed: %v", err)
		}

		if len(requests) != 0 {
			t.Errorf("Expected 0 requests, got %d", len(requests))
		}
	})
}

// TestColorRequestRepository_FindAllPending tests finding pending requests
func TestColorRequestRepository_FindAllPending(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewColorRequestRepository(db)

	user1ID := testutil.SeedTestUser(t, db, "user6@test.com", "User 6", "green")
	user2ID := testutil.SeedTestUser(t, db, "user7@test.com", "User 7", "green")
	color1ID := testutil.SeedTestColorCategory(t, db, "pending-color-1", "#333333", 50)
	color2ID := testutil.SeedTestColorCategory(t, db, "pending-color-2", "#444444", 60)

	// Create pending and non-pending requests
	testutil.SeedTestColorRequest(t, db, user1ID, color1ID, "pending")
	testutil.SeedTestColorRequest(t, db, user2ID, color2ID, "pending")
	testutil.SeedTestColorRequest(t, db, user1ID, color2ID, "approved")
	testutil.SeedTestColorRequest(t, db, user2ID, color1ID, "denied")

	t.Run("find only pending requests", func(t *testing.T) {
		requests, err := repo.FindAllPending()
		if err != nil {
			t.Fatalf("FindAllPending() failed: %v", err)
		}

		if len(requests) != 2 {
			t.Errorf("Expected 2 pending requests, got %d", len(requests))
		}

		// All should be pending
		for _, req := range requests {
			if req.Status != "pending" {
				t.Errorf("Expected status 'pending', got %s", req.Status)
			}
		}
	})
}

// TestColorRequestRepository_Approve tests approving requests
func TestColorRequestRepository_Approve(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewColorRequestRepository(db)

	userID := testutil.SeedTestUser(t, db, "user8@test.com", "User 8", "green")
	adminID := testutil.SeedTestUser(t, db, "admin@test.com", "Admin", "blue")
	colorID := testutil.SeedTestColorCategory(t, db, "approve-color", "#555555", 70)

	t.Run("successful approval", func(t *testing.T) {
		reqID := testutil.SeedTestColorRequest(t, db, userID, colorID, "pending")

		message := "Willkommen!"
		err := repo.Approve(reqID, adminID, &message)
		if err != nil {
			t.Fatalf("Approve() failed: %v", err)
		}

		// Verify approval
		request, _ := repo.FindByID(reqID)
		if request.Status != "approved" {
			t.Errorf("Expected status 'approved', got %s", request.Status)
		}
		if request.ReviewedBy == nil || *request.ReviewedBy != adminID {
			t.Error("ReviewedBy should be set to admin ID")
		}
		if request.AdminMessage == nil || *request.AdminMessage != message {
			t.Errorf("Expected message '%s', got %v", message, request.AdminMessage)
		}
		if request.ReviewedAt == nil {
			t.Error("ReviewedAt should be set")
		}
	})
}

// TestColorRequestRepository_Deny tests denying requests
func TestColorRequestRepository_Deny(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewColorRequestRepository(db)

	userID := testutil.SeedTestUser(t, db, "user9@test.com", "User 9", "green")
	adminID := testutil.SeedTestUser(t, db, "admin2@test.com", "Admin 2", "blue")
	colorID := testutil.SeedTestColorCategory(t, db, "deny-color", "#666666", 80)

	t.Run("successful denial", func(t *testing.T) {
		reqID := testutil.SeedTestColorRequest(t, db, userID, colorID, "pending")

		message := "Bitte erst Einweisung absolvieren"
		err := repo.Deny(reqID, adminID, &message)
		if err != nil {
			t.Fatalf("Deny() failed: %v", err)
		}

		// Verify denial
		request, _ := repo.FindByID(reqID)
		if request.Status != "denied" {
			t.Errorf("Expected status 'denied', got %s", request.Status)
		}
		if request.ReviewedBy == nil || *request.ReviewedBy != adminID {
			t.Error("ReviewedBy should be set to admin ID")
		}
	})
}

// TestColorRequestRepository_HasPendingRequest tests checking for pending requests
func TestColorRequestRepository_HasPendingRequest(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewColorRequestRepository(db)

	userID := testutil.SeedTestUser(t, db, "user10@test.com", "User 10", "green")
	color1ID := testutil.SeedTestColorCategory(t, db, "pending-check-1", "#777777", 90)
	color2ID := testutil.SeedTestColorCategory(t, db, "pending-check-2", "#888888", 100)

	t.Run("user has pending request", func(t *testing.T) {
		testutil.SeedTestColorRequest(t, db, userID, color1ID, "pending")

		hasPending, err := repo.HasPendingRequest(userID)
		if err != nil {
			t.Fatalf("HasPendingRequest() failed: %v", err)
		}

		if !hasPending {
			t.Error("Should have pending request")
		}
	})

	t.Run("user has no pending request", func(t *testing.T) {
		user2ID := testutil.SeedTestUser(t, db, "user11@test.com", "User 11", "green")
		testutil.SeedTestColorRequest(t, db, user2ID, color2ID, "approved")

		hasPending, err := repo.HasPendingRequest(user2ID)
		if err != nil {
			t.Fatalf("HasPendingRequest() failed: %v", err)
		}

		if hasPending {
			t.Error("Approved request should not count as pending")
		}
	})

	t.Run("user with denied request - not pending", func(t *testing.T) {
		user3ID := testutil.SeedTestUser(t, db, "user12@test.com", "User 12", "green")
		testutil.SeedTestColorRequest(t, db, user3ID, color1ID, "denied")

		hasPending, err := repo.HasPendingRequest(user3ID)
		if err != nil {
			t.Fatalf("HasPendingRequest() failed: %v", err)
		}

		if hasPending {
			t.Error("Denied request should not count as pending")
		}
	})
}
