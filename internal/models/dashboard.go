package models

// DashboardStats represents admin dashboard statistics
type DashboardStats struct {
	TotalWalksCompleted   int `json:"total_walks_completed"`
	UpcomingWalksToday    int `json:"upcoming_walks_today"`
	UpcomingWalksTotal    int `json:"upcoming_walks_total"`
	ActiveUsers           int `json:"active_users"`
	InactiveUsers         int `json:"inactive_users"`
	AvailableDogs         int `json:"available_dogs"`
	UnavailableDogs       int `json:"unavailable_dogs"`
	PendingColorRequests    int `json:"pending_color_requests"`
	PendingReactivationReqs int `json:"pending_reactivation_requests"`
}

// ActivityItem represents a recent activity item
type ActivityItem struct {
	Type      string `json:"type"`
	Message   string `json:"message"`
	Timestamp string `json:"timestamp"`
	UserID    *int   `json:"user_id,omitempty"`
	UserName  string `json:"user_name,omitempty"`
	DogID     *int   `json:"dog_id,omitempty"`
	DogName   string `json:"dog_name,omitempty"`
}

// RecentActivityResponse represents the recent activity feed
type RecentActivityResponse struct {
	Activities []*ActivityItem `json:"activities"`
}
