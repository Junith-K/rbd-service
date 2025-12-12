package models

import "time"

// FriendshipStatus represents the status of a friendship
type FriendshipStatus string

const (
	StatusPending  FriendshipStatus = "pending"
	StatusAccepted FriendshipStatus = "accepted"
	StatusRejected FriendshipStatus = "rejected"
)

// Friendship represents a friendship or friend request
type Friendship struct {
	FriendshipID         string           `firestore:"friendshipId" json:"friendshipId"`
	User1ID              string           `firestore:"user1Id" json:"user1Id"`
	User2ID              string           `firestore:"user2Id" json:"user2Id"`
	Status               FriendshipStatus `firestore:"status" json:"status"`
	RequestedAt          time.Time        `firestore:"requestedAt" json:"requestedAt"`
	AcceptedAt           *time.Time       `firestore:"acceptedAt,omitempty" json:"acceptedAt,omitempty"`
	User1Muted           bool             `firestore:"user1Muted" json:"user1Muted"`
	User2Muted           bool             `firestore:"user2Muted" json:"user2Muted"`
	User1CooldownMinutes int              `firestore:"user1CooldownMinutes" json:"user1CooldownMinutes"` // Cooldown User1 sets for User2
	User2CooldownMinutes int              `firestore:"user2CooldownMinutes" json:"user2CooldownMinutes"` // Cooldown User2 sets for User1
}

// FriendInfo represents friend information for display
type FriendInfo struct {
	UserID            string `json:"userId"`
	Username          string `json:"username"`
	IsMuted           bool   `json:"isMuted"`           // Have I muted this friend? (shows red button on my side)
	IsMutedBy         bool   `json:"isMutedBy"`         // Has this friend muted me? (disables my trigger button)
	CooldownRemaining int    `json:"cooldownRemaining"` // Remaining seconds until can trigger again
	CanTrigger        bool   `json:"canTrigger"`        // Whether user can trigger notification now
	CooldownMinutes   int    `json:"cooldownMinutes"`   // Cooldown duration in minutes set by current user for this friend
}

// FriendRequest represents a pending friend request
type FriendRequest struct {
	RequestID   string    `json:"requestId"`
	Username    string    `json:"username"`
	UserID      string    `json:"userId"`
	RequestedAt time.Time `json:"requestedAt"`
}

// SendFriendRequestBody represents the request body for sending friend request
type SendFriendRequestBody struct {
	TargetUserID string `json:"targetUserId" binding:"required"`
}

// AcceptRejectRequestBody represents the request body for accepting/rejecting friend request
type AcceptRejectRequestBody struct {
	RequestID string `json:"requestId" binding:"required"`
}

// SearchUsersRequest represents the search users request body
type SearchUsersRequest struct {
	Username string `json:"username" binding:"required"`
}

// MuteFriendRequest represents the mute friend request body
type MuteFriendRequest struct {
	FriendUserID string `json:"friendUserId" binding:"required"`
	Muted        bool   `json:"muted"`
}

// MuteAllRequest represents the mute all request body
type MuteAllRequest struct {
	MutedAll bool `json:"mutedAll"`
}

// UpdateCooldownRequest represents the update cooldown request body
type UpdateCooldownRequest struct {
	FriendUserID    string `json:"friendUserId" binding:"required"`
	CooldownMinutes int    `json:"cooldownMinutes" binding:"min=1,max=1440"` // 1 to 1440 minutes (1 day)
}
