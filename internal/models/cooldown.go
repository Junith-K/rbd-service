package models

import "time"

// Cooldown represents a cooldown period for notifications
type Cooldown struct {
	CooldownID   string    `firestore:"cooldownId" json:"cooldownId"`
	UserID       string    `firestore:"userId" json:"userId"`
	TargetUserID string    `firestore:"targetUserId" json:"targetUserId"`
	TriggeredAt  time.Time `firestore:"triggeredAt" json:"triggeredAt"`
	ExpiresAt    time.Time `firestore:"expiresAt" json:"expiresAt"`
}

// CooldownResponse represents the cooldown check response
type CooldownResponse struct {
	OnCooldown  bool       `json:"onCooldown"`
	AvailableAt *time.Time `json:"availableAt,omitempty"`
}
