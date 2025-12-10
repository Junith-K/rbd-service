package models

import "time"

// User represents a user in the system
type User struct {
	UserID       string    `firestore:"userId" json:"userId"`
	Username     string    `firestore:"username" json:"username"`
	PasswordHash string    `firestore:"passwordHash" json:"-"` // Don't expose in JSON
	FCMToken     string    `firestore:"fcmToken" json:"fcmToken,omitempty"`
	CreatedAt    time.Time `firestore:"createdAt" json:"createdAt"`
	MutedAll     bool      `firestore:"mutedAll" json:"mutedAll"`
}

// RegisterRequest represents the registration request body
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=16"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginRequest represents the login request body
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// AuthResponse represents the authentication response
type AuthResponse struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
	Token    string `json:"token"`
}

// UpdateFCMTokenRequest represents the FCM token update request
type UpdateFCMTokenRequest struct {
	FCMToken string `json:"fcmToken" binding:"required"`
}
