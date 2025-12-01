package repository

import (
	"testing"

	"github.com/tranmh/gassigeher/internal/models"
	"github.com/tranmh/gassigeher/internal/testutil"
)

// DONE: TestExperienceRequestRepository_Create tests experience request creation
func TestExperienceRequestRepository_Create(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewExperienceRequestRepository(db)

	// Create test user
	userID := testutil.SeedTestUser(t, db, "test@example.com", "Test User", "green")

	t.Run("successful creation", func(t *testing.T) {
		req := &models.ExperienceRequest{
			UserID:         userID,
			RequestedLevel: "blue",
			Status:         "pending",
		}

		err := repo.Create(req)
		if err != nil {
			t.Fatalf("Create() failed: %v", err)
		}

		if req.ID == 0 {
			t.Error("ExperienceRequest ID should be set after creation")
		}
	})
}

// DONE: TestExperienceRequestRepository_FindByID tests finding request by ID
func TestExperienceRequestRepository_FindByID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewExperienceRequestRepository(db)

	userID := testutil.SeedTestUser(t, db, "test@example.com", "Test User", "green")

	t.Run("request exists", func(t *testing.T) {
		reqID := testutil.SeedTestExperienceRequest(t, db, userID, "blue", "pending")

		request, err := repo.FindByID(reqID)
		if err != nil {
			t.Fatalf("FindByID() failed: %v", err)
		}

		if request.ID != reqID {
			t.Errorf("Expected ID %d, got %d", reqID, request.ID)
		}

		if request.RequestedLevel != "blue" {
			t.Errorf("Expected level 'blue', got %s", request.RequestedLevel)
		}
	})

	t.Run("request not found", func(t *testing.T) {
		request, _ := repo.FindByID(99999)
		if request != nil {
			t.Error("Expected nil for non-existent ID")
		}
	})
}

// DONE: TestExperienceRequestRepository_FindByUserID tests finding user's requests
func TestExperienceRequestRepository_FindByUserID(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewExperienceRequestRepository(db)

	user1ID := testutil.SeedTestUser(t, db, "user1@example.com", "User 1", "green")
	user2ID := testutil.SeedTestUser(t, db, "user2@example.com", "User 2", "green")

	// Create requests for user1
	testutil.SeedTestExperienceRequest(t, db, user1ID, "blue", "pending")
	testutil.SeedTestExperienceRequest(t, db, user1ID, "orange", "denied")

	// Create request for user2
	testutil.SeedTestExperienceRequest(t, db, user2ID, "blue", "pending")

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
		user3ID := testutil.SeedTestUser(t, db, "user3@example.com", "User 3", "green")

		requests, err := repo.FindByUserID(user3ID)
		if err != nil {
			t.Fatalf("FindByUserID() failed: %v", err)
		}

		if len(requests) != 0 {
			t.Errorf("Expected 0 requests, got %d", len(requests))
		}
	})
}

// DONE: TestExperienceRequestRepository_FindAllPending tests finding pending requests
func TestExperienceRequestRepository_FindAllPending(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewExperienceRequestRepository(db)

	user1ID := testutil.SeedTestUser(t, db, "user1@example.com", "User 1", "green")
	user2ID := testutil.SeedTestUser(t, db, "user2@example.com", "User 2", "blue")

	// Create pending and non-pending requests
	testutil.SeedTestExperienceRequest(t, db, user1ID, "blue", "pending")
	testutil.SeedTestExperienceRequest(t, db, user2ID, "orange", "pending")
	testutil.SeedTestExperienceRequest(t, db, user1ID, "orange", "approved")
	testutil.SeedTestExperienceRequest(t, db, user2ID, "blue", "denied")

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

// DONE: TestExperienceRequestRepository_Approve tests approving requests
func TestExperienceRequestRepository_Approve(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewExperienceRequestRepository(db)

	userID := testutil.SeedTestUser(t, db, "user@example.com", "User", "green")
	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "orange")

	t.Run("successful approval", func(t *testing.T) {
		reqID := testutil.SeedTestExperienceRequest(t, db, userID, "blue", "pending")

		message := "Well done!"
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

// DONE: TestExperienceRequestRepository_Deny tests denying requests
func TestExperienceRequestRepository_Deny(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewExperienceRequestRepository(db)

	userID := testutil.SeedTestUser(t, db, "user@example.com", "User", "green")
	adminID := testutil.SeedTestUser(t, db, "admin@example.com", "Admin", "orange")

	t.Run("successful denial", func(t *testing.T) {
		reqID := testutil.SeedTestExperienceRequest(t, db, userID, "blue", "pending")

		message := "Need more experience"
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

// DONE: TestExperienceRequestRepository_HasPendingRequest tests checking for pending requests
func TestExperienceRequestRepository_HasPendingRequest(t *testing.T) {
	db := testutil.SetupTestDB(t)
	repo := NewExperienceRequestRepository(db)

	userID := testutil.SeedTestUser(t, db, "user@example.com", "User", "green")

	t.Run("user has pending request for level", func(t *testing.T) {
		testutil.SeedTestExperienceRequest(t, db, userID, "blue", "pending")

		hasPending, err := repo.HasPendingRequest(userID, "blue")
		if err != nil {
			t.Fatalf("HasPendingRequest() failed: %v", err)
		}

		if !hasPending {
			t.Error("Should have pending request for blue level")
		}
	})

	t.Run("user has no pending request for level", func(t *testing.T) {
		hasPending, err := repo.HasPendingRequest(userID, "orange")
		if err != nil {
			t.Fatalf("HasPendingRequest() failed: %v", err)
		}

		if hasPending {
			t.Error("Should not have pending request for orange level")
		}
	})

	t.Run("user has approved request - not pending", func(t *testing.T) {
		user2ID := testutil.SeedTestUser(t, db, "user2@example.com", "User 2", "green")
		testutil.SeedTestExperienceRequest(t, db, user2ID, "blue", "approved")

		hasPending, err := repo.HasPendingRequest(user2ID, "blue")
		if err != nil {
			t.Fatalf("HasPendingRequest() failed: %v", err)
		}

		if hasPending {
			t.Error("Approved request should not count as pending")
		}
	})
}
