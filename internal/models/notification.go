package models

import "time"

// TriggerNotificationRequest represents the request to trigger a notification
type TriggerNotificationRequest struct {
	TargetUserID string `json:"targetUserId" binding:"required"`
}

// TriggerNotificationResponse represents the successful trigger response
type TriggerNotificationResponse struct {
	Success         bool      `json:"success"`
	NextAvailableAt time.Time `json:"nextAvailableAt"`
}

// TriggerErrorResponse represents an error response for trigger
type TriggerErrorResponse struct {
	Error       string     `json:"error"`
	AvailableAt *time.Time `json:"availableAt,omitempty"`
}
