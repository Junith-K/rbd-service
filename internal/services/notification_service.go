package services

import (
	"context"
	"errors"
	"fmt"
	"log"

	"firebase.google.com/go/messaging"
	"github.com/yourusername/rbd-service/internal/config"
	"github.com/yourusername/rbd-service/internal/models"
	"github.com/yourusername/rbd-service/internal/repository"
)

type NotificationService struct {
	userRepo     *repository.UserRepository
	friendRepo   *repository.FriendRepository
	cooldownRepo *repository.CooldownRepository
	historyRepo  *repository.HistoryRepository
}

func NewNotificationService() *NotificationService {
	return &NotificationService{
		userRepo:     repository.NewUserRepository(),
		friendRepo:   repository.NewFriendRepository(),
		cooldownRepo: repository.NewCooldownRepository(),
		historyRepo:  repository.NewHistoryRepository(),
	}
}

// TriggerNotification triggers a notification to a friend
func (s *NotificationService) TriggerNotification(ctx context.Context, senderID, targetUserID string) (*models.TriggerNotificationResponse, error) {
	// Get sender info
	sender, err := s.userRepo.GetUserByID(ctx, senderID)
	if err != nil {
		return nil, errors.New("sender not found")
	}

	// Get target user
	target, err := s.userRepo.GetUserByID(ctx, targetUserID)
	if err != nil {
		return nil, errors.New("target user not found")
	}

	// Check if users are friends
	friendship, err := s.friendRepo.CheckExistingFriendship(ctx, senderID, targetUserID)
	if err != nil {
		return nil, err
	}
	if friendship == nil || friendship.Status != models.StatusAccepted {
		return nil, errors.New("users are not friends")
	}

	// Check if TARGET has muted SENDER (target doesn't want to receive notifications from sender)
	targetMutedSender := false
	if friendship.User1ID == targetUserID {
		targetMutedSender = friendship.User1Muted // Target is User1, User1 muted sender (User2)
	} else {
		targetMutedSender = friendship.User2Muted // Target is User2, User2 muted sender (User1)
	}

	if targetMutedSender {
		return nil, errors.New("friend_muted_you")
	}

	// Check if target has muted all
	if target.MutedAll {
		return nil, errors.New("user_muted_all")
	}

	// Check cooldown
	activeCooldown, err := s.cooldownRepo.CheckActiveCooldown(ctx, senderID, targetUserID)
	if err != nil {
		return nil, err
	}
	if activeCooldown != nil {
		return nil, fmt.Errorf("cooldown_active:%s", activeCooldown.ExpiresAt.Format("2006-01-02T15:04:05Z07:00"))
	}

	// Get cooldown duration set by TARGET for SENDER
	// User1CooldownMinutes = how often User2 can trigger User1
	// User2CooldownMinutes = how often User1 can trigger User2
	cooldownMinutes := 60 // Default
	if friendship.User1ID == targetUserID {
		// Target is User1, sender is User2, use User1's cooldown setting for User2
		cooldownMinutes = friendship.User1CooldownMinutes
	} else {
		// Target is User2, sender is User1, use User2's cooldown setting for User1
		cooldownMinutes = friendship.User2CooldownMinutes
	}
	// Apply default for uninitialized friendships (<=0)
	if cooldownMinutes <= 0 {
		cooldownMinutes = 60
	}

	// Create cooldown with dynamic duration
	if err := s.cooldownRepo.CreateCooldown(ctx, senderID, targetUserID, cooldownMinutes); err != nil {
		return nil, err
	}

	// Create history record
	if err := s.historyRepo.CreateHistory(ctx, senderID, targetUserID, sender.Username); err != nil {
		log.Printf("Failed to create history: %v", err)
	}

	// Send FCM notification
	if target.FCMToken != "" {
		if err := s.sendFCMNotification(ctx, target.FCMToken, sender.Username, senderID); err != nil {
			log.Printf("⚠️ Failed to send FCM to %s: %v", targetUserID, err)
			// Note: If token is invalid, user needs to re-login to update it
		}
	} else {
		log.Printf("⚠️ Target user %s has no FCM token", targetUserID)
	}

	// Calculate next available time
	nextAvailable := activeCooldown
	if nextAvailable == nil {
		// Get the newly created cooldown
		newCooldown, _ := s.cooldownRepo.CheckActiveCooldown(ctx, senderID, targetUserID)
		if newCooldown != nil {
			nextAvailable = newCooldown
		}
	}

	response := &models.TriggerNotificationResponse{
		Success: true,
	}
	if nextAvailable != nil {
		response.NextAvailableAt = nextAvailable.ExpiresAt
	}

	return response, nil
}

// sendFCMNotification sends a push notification via FCM
func (s *NotificationService) sendFCMNotification(ctx context.Context, fcmToken, senderUsername, senderID string) error {
	client, err := config.FirebaseApp.Messaging(ctx)
	if err != nil {
		return fmt.Errorf("failed to get messaging client: %w", err)
	}

	message := &messaging.Message{
		Token: fcmToken,
		Notification: &messaging.Notification{
			Title: "Return By Death!",
			Body:  fmt.Sprintf("%s has called you back from death!", senderUsername),
		},
		Data: map[string]string{
			"type":           "respawn_trigger",
			"senderId":       senderID,
			"senderUsername": senderUsername,
		},
		Android: &messaging.AndroidConfig{
			Priority: "high",
			Notification: &messaging.AndroidNotification{
				ChannelID: "respawn_channel",
				Priority:  messaging.PriorityMax,
			},
		},
		APNS: &messaging.APNSConfig{
			Payload: &messaging.APNSPayload{
				Aps: &messaging.Aps{
					Sound: "respawn_sound.mp3",
					Badge: nil,
				},
			},
		},
	}

	_, err = client.Send(ctx, message)
	if err != nil {
		return fmt.Errorf("failed to send FCM: %w", err)
	}

	log.Printf("✅ Notification sent to token: %s", fcmToken[:20]+"...")
	return nil
}

// CheckCooldown checks if there's an active cooldown
func (s *NotificationService) CheckCooldown(ctx context.Context, senderID, targetUserID string) (*models.CooldownResponse, error) {
	cooldown, err := s.cooldownRepo.CheckActiveCooldown(ctx, senderID, targetUserID)
	if err != nil {
		return nil, err
	}

	if cooldown == nil {
		return &models.CooldownResponse{
			OnCooldown:  false,
			AvailableAt: nil,
		}, nil
	}

	return &models.CooldownResponse{
		OnCooldown:  true,
		AvailableAt: &cooldown.ExpiresAt,
	}, nil
}
