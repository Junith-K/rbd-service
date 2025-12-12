package services

import (
	"context"
	"errors"
	"time"

	"github.com/yourusername/rbd-service/internal/models"
	"github.com/yourusername/rbd-service/internal/repository"
)

type FriendService struct {
	friendRepo   *repository.FriendRepository
	userRepo     *repository.UserRepository
	historyRepo  *repository.HistoryRepository
	cooldownRepo *repository.CooldownRepository
}

func NewFriendService() *FriendService {
	return &FriendService{
		friendRepo:   repository.NewFriendRepository(),
		userRepo:     repository.NewUserRepository(),
		historyRepo:  repository.NewHistoryRepository(),
		cooldownRepo: repository.NewCooldownRepository(),
	}
}

// GetFriends returns all accepted friends for a user
func (s *FriendService) GetFriends(ctx context.Context, userID string) ([]*models.FriendInfo, error) {
	friendships, err := s.friendRepo.GetAcceptedFriends(ctx, userID)
	if err != nil {
		return nil, err
	}

	var friends []*models.FriendInfo
	for _, friendship := range friendships {
		// Determine which user is the friend and check if THEY muted ME
		// isMuted means: "Has this friend muted me? (Can I trigger them?)"
		friendUserID := friendship.User2ID
		isMuted := friendship.User2Muted // User2 (friend) muted User1 (me)
		cooldownMinutes := friendship.User2CooldownMinutes // User2's cooldown for User1 (how often User1 can trigger User2)
		isUser1 := true
		
		if friendship.User2ID == userID {
			friendUserID = friendship.User1ID
			isMuted = friendship.User1Muted // User1 (friend) muted User2 (me)
			cooldownMinutes = friendship.User1CooldownMinutes // User1's cooldown for User2 (how often User2 can trigger User1)
			isUser1 = false
		}
		
		// Apply default cooldown for old or uninitialized friendships
		// 0 = uninitialized (old friendships or missing field)
		// Minimum valid cooldown is 1 minute
		if cooldownMinutes <= 0 {
			cooldownMinutes = 60 // Default to 60 minutes
			// Update DB (best effort, don't fail if error)
			_ = s.friendRepo.UpdateCooldown(ctx, friendship.FriendshipID, isUser1, 60)
		}

		// Get friend's username
		user, err := s.userRepo.GetUserByID(ctx, friendUserID)
		if err != nil {
			continue // Skip if user not found
		}

		// Check cooldown status - current user (userID) trying to trigger friend (friendUserID)
		cooldown, err := s.cooldownRepo.CheckActiveCooldown(ctx, userID, friendUserID)
		
		cooldownRemaining := 0
		canTrigger := true
		
		if err == nil && cooldown != nil {
			// Calculate remaining seconds
			now := time.Now()
			remaining := cooldown.ExpiresAt.Sub(now)
			if remaining > 0 {
				cooldownRemaining = int(remaining.Seconds())
				canTrigger = false
			}
		}

		friends = append(friends, &models.FriendInfo{
			UserID:            friendUserID,
			Username:          user.Username,
			IsMuted:           isMuted,
			CooldownRemaining: cooldownRemaining,
			CanTrigger:        canTrigger,
			CooldownMinutes:   cooldownMinutes,
		})
	}

	return friends, nil
}

// GetPendingRequests returns pending friend requests for a user
func (s *FriendService) GetPendingRequests(ctx context.Context, userID string) ([]*models.FriendRequest, error) {
	friendships, err := s.friendRepo.GetPendingRequests(ctx, userID)
	if err != nil {
		return nil, err
	}

	var requests []*models.FriendRequest
	for _, friendship := range friendships {
		// Get requester's username
		user, err := s.userRepo.GetUserByID(ctx, friendship.User1ID)
		if err != nil {
			continue
		}

		requests = append(requests, &models.FriendRequest{
			RequestID:   friendship.FriendshipID,
			Username:    user.Username,
			UserID:      friendship.User1ID,
			RequestedAt: friendship.RequestedAt,
		})
	}

	return requests, nil
}

// SearchUsers searches for users by username
func (s *FriendService) SearchUsers(ctx context.Context, currentUserID, searchUsername string) ([]*models.FriendInfo, error) {
	users, err := s.userRepo.SearchUsersByUsername(ctx, searchUsername, 20)
	if err != nil {
		return nil, err
	}

	var results []*models.FriendInfo
	for _, user := range users {
		// Don't include current user in results
		if user.UserID == currentUserID {
			continue
		}

		// Check if username contains search term (case-insensitive would be better)
		results = append(results, &models.FriendInfo{
			UserID:   user.UserID,
			Username: user.Username,
			IsMuted:  false,
		})
	}

	return results, nil
}

// SendFriendRequest sends a friend request
func (s *FriendService) SendFriendRequest(ctx context.Context, senderID, targetUserID string) (string, error) {
	// Check if users are the same
	if senderID == targetUserID {
		return "", errors.New("cannot send friend request to yourself")
	}

	// Check if target user exists
	_, err := s.userRepo.GetUserByID(ctx, targetUserID)
	if err != nil {
		return "", errors.New("user not found")
	}

	// Check if friendship already exists (check both directions to prevent race condition)
	existing, err := s.friendRepo.CheckExistingFriendship(ctx, senderID, targetUserID)
	if err != nil {
		return "", err
	}
	if existing != nil {
		if existing.Status == models.StatusAccepted {
			return "", errors.New("already friends")
		}
		if existing.Status == models.StatusPending {
			// Check who sent the original request
			if existing.User1ID == senderID {
				return "", errors.New("friend request already sent")
			} else {
				return "", errors.New("this user already sent you a friend request")
			}
		}
	}

	// Create friend request
	requestID, err := s.friendRepo.CreateFriendRequest(ctx, senderID, targetUserID)
	if err != nil {
		return "", err
	}

	return requestID, nil
}

// AcceptFriendRequest accepts a friend request
func (s *FriendService) AcceptFriendRequest(ctx context.Context, userID, requestID string) error {
	// Get friendship
	friendship, err := s.friendRepo.GetFriendship(ctx, requestID)
	if err != nil {
		return errors.New("friend request not found")
	}

	// Verify user is the recipient
	if friendship.User2ID != userID {
		return errors.New("unauthorized to accept this request")
	}

	// Verify status is pending
	if friendship.Status != models.StatusPending {
		return errors.New("friend request is not pending")
	}

	// Accept request
	return s.friendRepo.AcceptFriendRequest(ctx, requestID)
}

// RejectFriendRequest rejects a friend request
func (s *FriendService) RejectFriendRequest(ctx context.Context, userID, requestID string) error {
	// Get friendship
	friendship, err := s.friendRepo.GetFriendship(ctx, requestID)
	if err != nil {
		return errors.New("friend request not found")
	}

	// Verify user is the recipient
	if friendship.User2ID != userID {
		return errors.New("unauthorized to reject this request")
	}

	// Verify status is pending
	if friendship.Status != models.StatusPending {
		return errors.New("friend request is not pending")
	}

	// Reject request
	return s.friendRepo.RejectFriendRequest(ctx, requestID)
}

// RemoveFriend removes a friendship
func (s *FriendService) RemoveFriend(ctx context.Context, userID, friendUserID string) error {
	// Find friendship
	existing, err := s.friendRepo.CheckExistingFriendship(ctx, userID, friendUserID)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("friendship not found")
	}

	// Delete friendship
	return s.friendRepo.DeleteFriendship(ctx, existing.FriendshipID)
}

// MuteFriend mutes or unmutes a friend
func (s *FriendService) MuteFriend(ctx context.Context, userID, friendUserID string, muted bool) error {
	// Find friendship
	existing, err := s.friendRepo.CheckExistingFriendship(ctx, userID, friendUserID)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("friendship not found")
	}

	// Determine if current user is User1 or User2
	isUser1 := existing.User1ID == userID

	return s.friendRepo.UpdateMuteStatus(ctx, existing.FriendshipID, isUser1, muted)
}

// MuteAll mutes or unmutes all friends
func (s *FriendService) MuteAll(ctx context.Context, userID string, mutedAll bool) error {
	return s.userRepo.UpdateMuteAll(ctx, userID, mutedAll)
}

// UpdateFriendCooldown updates the cooldown duration for a specific friend
func (s *FriendService) UpdateFriendCooldown(ctx context.Context, userID, friendUserID string, cooldownMinutes int) error {
	// Validate cooldown range (1 to 1440 minutes = 1 day)
	// Minimum 1 minute to avoid ambiguity with uninitialized state
	if cooldownMinutes < 1 || cooldownMinutes > 1440 {
		return errors.New("cooldown must be between 1 and 1440 minutes")
	}

	// Find friendship
	existing, err := s.friendRepo.CheckExistingFriendship(ctx, userID, friendUserID)
	if err != nil {
		return err
	}
	if existing == nil {
		return errors.New("friendship not found")
	}

	// Verify friendship is accepted
	if existing.Status != models.StatusAccepted {
		return errors.New("can only set cooldown for accepted friends")
	}

	// Determine if current user is User1 or User2
	isUser1 := existing.User1ID == userID

	// Update the cooldown setting
	if err := s.friendRepo.UpdateCooldown(ctx, existing.FriendshipID, isUser1, cooldownMinutes); err != nil {
		return err
	}

	// Update any active cooldown to use the new duration
	// Note: This updates the friendUserID->userID cooldown (friend triggering current user)
	// We need to update userID->friendUserID cooldown (current user triggering friend)
	_, _ = s.cooldownRepo.UpdateActiveCooldown(ctx, userID, friendUserID, cooldownMinutes)

	return nil
}
