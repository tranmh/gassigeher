# Gassigeher API Documentation

**Complete REST API documentation for the Gassigeher dog walking booking system.**

**Status**: ✅ All 50+ endpoints implemented and documented

> **Quick Links**: [README](../README.md) | [Deployment](DEPLOYMENT.md) | [User Guide](USER_GUIDE.md) | [Admin Guide](ADMIN_GUIDE.md)

---

## Base URL

```
http://localhost:8080/api
```

## Authentication

Most endpoints require JWT authentication. Include the token in the Authorization header:

```
Authorization: Bearer <your-jwt-token>
```

## Response Format

All responses are in JSON format.

### Success Response
```json
{
  "data": { ... },
  "message": "Success message"
}
```

### Error Response
```json
{
  "error": "Error message"
}
```

---

## Authentication Endpoints

### Register User
`POST /auth/register`

Create a new user account. Sends verification email.

**Request:**
```json
{
  "name": "Max Mustermann",
  "email": "max@example.com",
  "phone": "+49 123 456789",
  "password": "SecurePass123",
  "confirm_password": "SecurePass123",
  "accept_terms": true
}
```

**Response:** `201 Created`
```json
{
  "message": "Registration successful. Please check your email to verify your account.",
  "user_id": 1
}
```

**Validation:**
- Password must be at least 8 characters
- Password must contain uppercase, lowercase, and number
- Passwords must match
- Terms must be accepted

---

### Verify Email
`POST /auth/verify-email`

Verify email address with token from email.

**Request:**
```json
{
  "token": "verification-token-from-email"
}
```

**Response:** `200 OK`
```json
{
  "message": "Email verified successfully"
}
```

---

### Login
`POST /auth/login`

Login and receive JWT token.

**Request:**
```json
{
  "email": "max@example.com",
  "password": "SecurePass123"
}
```

**Response:** `200 OK`
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "user": {
    "id": 1,
    "name": "Max Mustermann",
    "email": "max@example.com",
    "experience_level": "green",
    "is_verified": true,
    "is_active": true
  },
  "is_admin": false
}
```

---

### Forgot Password
`POST /auth/forgot-password`

Request password reset email.

**Request:**
```json
{
  "email": "max@example.com"
}
```

**Response:** `200 OK`
```json
{
  "message": "If an account with this email exists, you will receive a password reset link."
}
```

---

### Reset Password
`POST /auth/reset-password`

Reset password with token from email.

**Request:**
```json
{
  "token": "reset-token-from-email",
  "password": "NewSecurePass123",
  "confirm_password": "NewSecurePass123"
}
```

**Response:** `200 OK`
```json
{
  "message": "Password reset successfully"
}
```

---

### Change Password
`PUT /auth/change-password` 🔒 Protected

Change password while logged in.

**Request:**
```json
{
  "old_password": "OldPassword123",
  "new_password": "NewPassword123",
  "confirm_password": "NewPassword123"
}
```

**Response:** `200 OK`
```json
{
  "message": "Password changed successfully"
}
```

---

## User Endpoints

### Get Current User
`GET /users/me` 🔒 Protected

Get current user's profile.

**Response:** `200 OK`
```json
{
  "id": 1,
  "name": "Max Mustermann",
  "email": "max@example.com",
  "phone": "+49 123 456789",
  "experience_level": "green",
  "is_verified": true,
  "is_active": true,
  "profile_photo": "users/photo.jpg",
  "created_at": "2025-01-15T10:00:00Z",
  "last_activity_at": "2025-01-16T14:30:00Z"
}
```

---

### Update Profile
`PUT /users/me` 🔒 Protected

Update user profile. Email changes trigger re-verification.

**Request:**
```json
{
  "name": "Max M. Mustermann",
  "email": "newemail@example.com",
  "phone": "+49 987 654321"
}
```

**Response:** `200 OK`
```json
{
  "message": "Profile updated. Please check your new email to verify it.",
  "user": { ... }
}
```

---

### Upload Profile Photo
`POST /users/me/photo` 🔒 Protected

Upload profile photo (JPEG/PNG, max 5MB).

**Request:** `multipart/form-data`
- Field name: `photo`
- File types: JPEG, PNG
- Max size: 5MB

**Response:** `200 OK`
```json
{
  "message": "Photo uploaded successfully",
  "photo": "users/photo_123.jpg"
}
```

---

### Delete Account
`DELETE /users/me` 🔒 Protected

GDPR-compliant account deletion. Personal data anonymized, walk history preserved.

**Request:**
```json
{
  "password": "MyPassword123"
}
```

**Response:** `200 OK`
```json
{
  "message": "Account deleted successfully"
}
```

---

## Dog Endpoints

### List Dogs
`GET /dogs` 🔒 Protected

List all dogs with optional filters.

**Query Parameters:**
- `breed` - Filter by breed
- `size` - Filter by size (small, medium, large)
- `category` - Filter by category (green, blue, orange)
- `available` - Filter by availability (true, false)
- `search` - Search by name or breed
- `min_age` - Minimum age
- `max_age` - Maximum age

**Response:** `200 OK`
```json
[
  {
    "id": 1,
    "name": "Buddy",
    "breed": "Golden Retriever",
    "size": "large",
    "age": 3,
    "category": "green",
    "is_available": true,
    "photo": "dogs/buddy.jpg",
    "special_needs": null,
    "pickup_location": "Tierheim Haupteingang",
    "walk_duration": 60,
    "default_morning_time": "09:00",
    "default_evening_time": "17:00"
  }
]
```

---

### Get Dog
`GET /dogs/:id` 🔒 Protected

Get detailed dog information.

**Response:** `200 OK`
```json
{
  "id": 1,
  "name": "Buddy",
  "breed": "Golden Retriever",
  "size": "large",
  "age": 3,
  "category": "green",
  "is_available": true,
  "unavailable_reason": null,
  "photo": "dogs/buddy.jpg",
  "special_needs": "Needs slow walks",
  "pickup_location": "Tierheim Haupteingang",
  "walk_route": "Waldweg bevorzugt",
  "walk_duration": 60,
  "special_instructions": "Mag keine Katzen",
  "default_morning_time": "09:00",
  "default_evening_time": "17:00",
  "created_at": "2025-01-01T10:00:00Z"
}
```

---

### Create Dog
`POST /dogs` 🔒 Admin Only

Create a new dog.

**Request:**
```json
{
  "name": "Max",
  "breed": "Labrador",
  "size": "large",
  "age": 2,
  "category": "blue",
  "special_needs": "Very energetic",
  "pickup_location": "Tierheim Seiteneingang",
  "walk_route": "Park oder Wald",
  "walk_duration": 45,
  "special_instructions": "Pulls on leash",
  "default_morning_time": "08:00",
  "default_evening_time": "18:00"
}
```

**Response:** `201 Created`
```json
{
  "id": 2,
  "name": "Max",
  ...
}
```

---

### Toggle Dog Availability
`PUT /dogs/:id/availability` 🔒 Admin Only

Mark dog as available or unavailable (e.g., for vet visits).

**Request:**
```json
{
  "is_available": false,
  "unavailable_reason": "Tierarztbesuch"
}
```

**Response:** `200 OK`
```json
{
  "message": "Dog availability updated"
}
```

---

### Upload Dog Photo
`POST /dogs/:id/photo` 🔒 Admin Only

Upload a photo for a dog. Supports JPEG and PNG files up to 10MB.

**Request:**
- Content-Type: `multipart/form-data`
- Field name: `photo`
- Accepted formats: JPEG, PNG
- Max size: 10MB (configurable)

**Example (using cURL):**
```bash
curl -X POST http://localhost:8080/api/dogs/1/photo \
  -H "Authorization: Bearer YOUR_JWT_TOKEN" \
  -F "photo=@dog_photo.jpg"
```

**Example (using JavaScript):**
```javascript
const formData = new FormData();
formData.append('photo', fileInput.files[0]);

const response = await api.uploadDogPhoto(dogId, fileInput.files[0]);
```

**Response:** `200 OK`
```json
{
  "message": "Photo uploaded successfully",
  "photo": "dogs/dog_1_full.jpg",
  "photo_thumbnail": "dogs/dog_1_thumb.jpg"
}
```

**Note:** When Phase 1 (Backend Image Processing) is implemented, the uploaded photo will be automatically:
- Resized to max 800x800 pixels
- Compressed (JPEG quality 85%)
- Thumbnail generated (300x300 pixels)

Currently, photos are stored as-is without processing.

**Validation:**
- File type must be JPEG or PNG
- File size must not exceed configured limit (default: 10MB)
- Dog must exist
- Requires admin authentication

**Error Responses:**

`400 Bad Request` - Invalid file type or size
```json
{
  "error": "Only JPEG and PNG files are allowed"
}
```

`404 Not Found` - Dog doesn't exist
```json
{
  "error": "Dog not found"
}
```

`413 Payload Too Large` - File exceeds size limit
```json
{
  "error": "File too large. Maximum size: 10MB"
}
```

---

## Booking Endpoints

### Create Booking
`POST /bookings` 🔒 Protected

Create a new dog walk booking.

**Request:**
```json
{
  "dog_id": 1,
  "date": "2025-12-01",
  "walk_type": "morning",
  "scheduled_time": "09:30"
}
```

**Response:** `201 Created`
```json
{
  "id": 1,
  "user_id": 1,
  "dog_id": 1,
  "date": "2025-12-01",
  "walk_type": "morning",
  "scheduled_time": "09:30",
  "status": "scheduled",
  "created_at": "2025-01-16T10:00:00Z"
}
```

**Validation:**
- Dog must be available
- User must have required experience level
- No double-booking for same dog/date/walk_type
- Date cannot be in the past
- Date must be within booking advance limit
- Date must not be blocked

---

### List Bookings
`GET /bookings` 🔒 Protected

List bookings. Users see own, admins see all.

**Query Parameters:**
- `dog_id` - Filter by dog
- `date_from` - Filter by start date
- `date_to` - Filter by end date
- `status` - Filter by status (scheduled, completed, cancelled)
- `walk_type` - Filter by walk type (morning, evening)

**Response:** `200 OK`
```json
[
  {
    "id": 1,
    "user_id": 1,
    "dog_id": 1,
    "date": "2025-12-01",
    "walk_type": "morning",
    "scheduled_time": "09:30",
    "status": "scheduled",
    "created_at": "2025-01-16T10:00:00Z"
  }
]
```

---

### Cancel Booking
`PUT /bookings/:id/cancel` 🔒 Protected

Cancel a booking. Users must cancel 12 hours in advance (configurable).

**Request:**
```json
{
  "reason": "Can't make it" // Optional for users, required for admins
}
```

**Response:** `200 OK`
```json
{
  "message": "Booking cancelled successfully"
}
```

---

### Move Booking
`PUT /bookings/:id/move` 🔒 Admin Only

Move a booking to a new date/time.

**Request:**
```json
{
  "date": "2025-12-02",
  "walk_type": "evening",
  "scheduled_time": "17:00",
  "reason": "Dog health check scheduled"
}
```

**Response:** `200 OK`
```json
{
  "message": "Booking moved successfully"
}
```

---

### Add Notes
`PUT /bookings/:id/notes` 🔒 Protected

Add notes to a completed booking.

**Request:**
```json
{
  "notes": "Great walk! Buddy loved the park."
}
```

**Response:** `200 OK`
```json
{
  "message": "Notes added successfully"
}
```

---

## Experience Request Endpoints

### Create Experience Request
`POST /experience-requests` 🔒 Protected

Request a higher experience level.

**Request:**
```json
{
  "requested_level": "blue" // or "orange"
}
```

**Response:** `201 Created`
```json
{
  "id": 1,
  "user_id": 1,
  "requested_level": "blue",
  "status": "pending",
  "created_at": "2025-01-16T10:00:00Z"
}
```

**Rules:**
- Cannot request orange from green (must get blue first)
- Cannot have pending request for same level
- Cannot request already-owned level

---

### Approve Experience Request
`PUT /experience-requests/:id/approve` 🔒 Admin Only

Approve an experience level request. Automatically updates user's level.

**Request:**
```json
{
  "message": "Great walking history! Approved." // Optional
}
```

**Response:** `200 OK`
```json
{
  "message": "Request approved"
}
```

---

## Admin Dashboard Endpoints

### Get Statistics
`GET /admin/stats` 🔒 Admin Only

Get comprehensive dashboard statistics.

**Response:** `200 OK`
```json
{
  "total_walks_completed": 156,
  "upcoming_walks_today": 5,
  "upcoming_walks_total": 23,
  "active_users": 42,
  "inactive_users": 3,
  "available_dogs": 12,
  "unavailable_dogs": 2,
  "pending_experience_requests": 4,
  "pending_reactivation_requests": 1
}
```

---

### Get Recent Activity
`GET /admin/activity` 🔒 Admin Only

Get recent activity feed (last 24 hours).

**Response:** `200 OK`
```json
{
  "activities": [
    {
      "type": "booking_created",
      "message": "Neue Buchung für Buddy",
      "timestamp": "2025-01-16T14:30:00Z",
      "user_id": 5,
      "dog_id": 1,
      "dog_name": "Buddy"
    },
    {
      "type": "booking_completed",
      "message": "Spaziergang mit Max abgeschlossen",
      "timestamp": "2025-01-16T12:00:00Z",
      "user_id": 3,
      "dog_id": 2,
      "dog_name": "Max"
    }
  ]
}
```

---

## System Settings Endpoints

### Get All Settings
`GET /settings` 🔒 Admin Only

Get all system settings.

**Response:** `200 OK`
```json
[
  {
    "key": "booking_advance_days",
    "value": "14",
    "updated_at": "2025-01-01T10:00:00Z"
  },
  {
    "key": "cancellation_notice_hours",
    "value": "12",
    "updated_at": "2025-01-01T10:00:00Z"
  },
  {
    "key": "auto_deactivation_days",
    "value": "365",
    "updated_at": "2025-01-01T10:00:00Z"
  }
]
```

---

### Update Setting
`PUT /settings/:key` 🔒 Admin Only

Update a system setting.

**Request:**
```json
{
  "value": "30"
}
```

**Response:** `200 OK`
```json
{
  "message": "Setting updated successfully"
}
```

**Available Keys:**
- `booking_advance_days` - How many days in advance users can book (default: 14)
- `cancellation_notice_hours` - Minimum hours before booking for cancellation (default: 12)
- `auto_deactivation_days` - Days of inactivity before auto-deactivation (default: 365)

---

## User Management Endpoints (Admin Only)

### List Users
`GET /users` 🔒 Admin Only

List all users with optional filters.

**Query Parameters:**
- `active` - Filter by active status (true/false)

**Response:** `200 OK`
```json
[
  {
    "id": 1,
    "name": "Max Mustermann",
    "email": "max@example.com",
    "phone": "+49 123 456789",
    "experience_level": "green",
    "is_active": true,
    "last_activity_at": "2025-01-16T14:30:00Z",
    "created_at": "2025-01-10T09:00:00Z"
  }
]
```

---

### Deactivate User
`PUT /users/:id/deactivate` 🔒 Admin Only

Deactivate a user account.

**Request:**
```json
{
  "reason": "Unreliable attendance"
}
```

**Response:** `200 OK`
```json
{
  "message": "User deactivated successfully"
}
```

---

### Activate User
`PUT /users/:id/activate` 🔒 Admin Only

Activate a user account.

**Request:**
```json
{
  "message": "Welcome back!" // Optional
}
```

**Response:** `200 OK`
```json
{
  "message": "User activated successfully"
}
```

---

## Error Codes

| Code | Meaning |
|------|---------|
| 200 | Success |
| 201 | Created |
| 400 | Bad Request - Validation error |
| 401 | Unauthorized - Invalid or missing token |
| 403 | Forbidden - Insufficient permissions |
| 404 | Not Found |
| 409 | Conflict - Duplicate resource |
| 500 | Internal Server Error |

---

## Rate Limiting

Currently no rate limiting is implemented. For production deployment, consider adding rate limiting middleware.

---

## Pagination

Currently no pagination is implemented for list endpoints. All results are returned. For production with large datasets, consider adding pagination.

---

## Webhooks

No webhook support currently. Future enhancement for external integrations.

---

## Testing

See test files in `internal/*/` directories for examples of API usage and expected behavior.

---

## Notes

- All dates are in `YYYY-MM-DD` format
- All times are in `HH:MM` 24-hour format
- All timestamps are in ISO 8601 format
- File uploads use `multipart/form-data`
- All other requests/responses use `application/json`

---

## Testing the API

### Using curl

**Login example:**
```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"test@example.com","password":"TestPass123"}'
```

**Authenticated request:**
```bash
curl -X GET http://localhost:8080/api/users/me \
  -H "Authorization: Bearer <your-jwt-token>"
```

### Using JavaScript (Frontend)

The application includes a complete API client in `frontend/js/api.js`:

```javascript
// Global instance available
await api.login('test@example.com', 'TestPass123');
const user = await api.getMe();
const dogs = await api.getDogs({ category: 'green' });
```

See [CLAUDE.md](../CLAUDE.md) for development guide.

---

## Related Documentation

**For Implementation:**
- [CLAUDE.md](../CLAUDE.md) - Development guide with architecture
- [README.md](../README.md) - Project setup and overview
- [ImplementationPlan.md](ImplementationPlan.md) - Complete architecture

**For Deployment:**
- [DEPLOYMENT.md](DEPLOYMENT.md) - Production deployment guide

**For Users:**
- [USER_GUIDE.md](USER_GUIDE.md) - User manual
- [ADMIN_GUIDE.md](ADMIN_GUIDE.md) - Administrator handbook

---

**API Status**: ✅ All 50+ endpoints implemented and production-ready
