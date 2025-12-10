package repository

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/yourusername/rbd-service/internal/config"
	"github.com/yourusername/rbd-service/internal/models"
	"google.golang.org/api/iterator"
)

type FriendRepository struct {
	client *firestore.Client
}

func NewFriendRepository() *FriendRepository {
	return &FriendRepository{
		client: config.FirestoreClient,
	}
}

// CreateFriendRequest creates a new friend request
func (r *FriendRepository) CreateFriendRequest(ctx context.Context, user1ID, user2ID string) (string, error) {
	friendship := models.Friendship{
		User1ID:              user1ID,
		User2ID:              user2ID,
		Status:               models.StatusPending,
		RequestedAt:          time.Now(),
		User1Muted:           false,
		User2Muted:           false,
		User1CooldownMinutes: 60, // Default 60 minutes for new friendships
		User2CooldownMinutes: 60, // Default 60 minutes for new friendships
	}

	docRef, _, err := r.client.Collection("friends").Add(ctx, friendship)
	if err != nil {
		return "", err
	}

	// Update with the generated ID
	friendshipID := docRef.ID
	_, err = docRef.Update(ctx, []firestore.Update{
		{Path: "friendshipId", Value: friendshipID},
	})
	if err != nil {
		return "", err
	}

	return friendshipID, nil
}

// GetFriendship retrieves a friendship by ID
func (r *FriendRepository) GetFriendship(ctx context.Context, friendshipID string) (*models.Friendship, error) {
	doc, err := r.client.Collection("friends").Doc(friendshipID).Get(ctx)
	if err != nil {
		return nil, err
	}

	var friendship models.Friendship
	if err := doc.DataTo(&friendship); err != nil {
		return nil, err
	}

	return &friendship, nil
}

// GetAcceptedFriends retrieves all accepted friends for a user
func (r *FriendRepository) GetAcceptedFriends(ctx context.Context, userID string) ([]*models.Friendship, error) {
	var friendships []*models.Friendship

	// Query where user is user1
	iter1 := r.client.Collection("friends").
		Where("user1Id", "==", userID).
		Where("status", "==", string(models.StatusAccepted)).
		Documents(ctx)

	for {
		doc, err := iter1.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var friendship models.Friendship
		if err := doc.DataTo(&friendship); err != nil {
			continue
		}
		friendships = append(friendships, &friendship)
	}

	// Query where user is user2
	iter2 := r.client.Collection("friends").
		Where("user2Id", "==", userID).
		Where("status", "==", string(models.StatusAccepted)).
		Documents(ctx)

	for {
		doc, err := iter2.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var friendship models.Friendship
		if err := doc.DataTo(&friendship); err != nil {
			continue
		}
		friendships = append(friendships, &friendship)
	}

	return friendships, nil
}

// GetPendingRequests retrieves pending friend requests for a user (where they are user2)
func (r *FriendRepository) GetPendingRequests(ctx context.Context, userID string) ([]*models.Friendship, error) {
	var friendships []*models.Friendship

	iter := r.client.Collection("friends").
		Where("user2Id", "==", userID).
		Where("status", "==", string(models.StatusPending)).
		Documents(ctx)

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var friendship models.Friendship
		if err := doc.DataTo(&friendship); err != nil {
			continue
		}
		friendships = append(friendships, &friendship)
	}

	return friendships, nil
}

// AcceptFriendRequest accepts a friend request
func (r *FriendRepository) AcceptFriendRequest(ctx context.Context, friendshipID string) error {
	now := time.Now()
	_, err := r.client.Collection("friends").Doc(friendshipID).Update(ctx, []firestore.Update{
		{Path: "status", Value: string(models.StatusAccepted)},
		{Path: "acceptedAt", Value: now},
	})
	return err
}

// RejectFriendRequest rejects a friend request
func (r *FriendRepository) RejectFriendRequest(ctx context.Context, friendshipID string) error {
	_, err := r.client.Collection("friends").Doc(friendshipID).Update(ctx, []firestore.Update{
		{Path: "status", Value: string(models.StatusRejected)},
	})
	return err
}

// DeleteFriendship deletes a friendship
func (r *FriendRepository) DeleteFriendship(ctx context.Context, friendshipID string) error {
	_, err := r.client.Collection("friends").Doc(friendshipID).Delete(ctx)
	return err
}

// UpdateMuteStatus updates the mute status for a friendship
func (r *FriendRepository) UpdateMuteStatus(ctx context.Context, friendshipID string, isUser1 bool, muted bool) error {
	fieldName := "user2Muted"
	if isUser1 {
		fieldName = "user1Muted"
	}
	_, err := r.client.Collection("friends").Doc(friendshipID).Update(ctx, []firestore.Update{
		{Path: fieldName, Value: muted},
	})
	return err
}

// UpdateCooldown updates the cooldown minutes for a friendship
func (r *FriendRepository) UpdateCooldown(ctx context.Context, friendshipID string, isUser1 bool, cooldownMinutes int) error {
	// When User1 sets cooldown for User2, update User2CooldownMinutes
	// (User2's setting for how often User1 can trigger User2)
	fieldName := "user2CooldownMinutes"
	if !isUser1 {
		fieldName = "user1CooldownMinutes"
	}
	_, err := r.client.Collection("friends").Doc(friendshipID).Update(ctx, []firestore.Update{
		{Path: fieldName, Value: cooldownMinutes},
	})
	return err
}

// CheckExistingFriendship checks if a friendship already exists between two users
func (r *FriendRepository) CheckExistingFriendship(ctx context.Context, user1ID, user2ID string) (*models.Friendship, error) {
	// Check both directions
	iter := r.client.Collection("friends").
		Where("user1Id", "==", user1ID).
		Where("user2Id", "==", user2ID).
		Limit(1).
		Documents(ctx)

	doc, err := iter.Next()
	if err == nil {
		var friendship models.Friendship
		if err := doc.DataTo(&friendship); err == nil {
			return &friendship, nil
		}
	}

	// Check reverse direction
	iter2 := r.client.Collection("friends").
		Where("user1Id", "==", user2ID).
		Where("user2Id", "==", user1ID).
		Limit(1).
		Documents(ctx)

	doc2, err2 := iter2.Next()
	if err2 == nil {
		var friendship models.Friendship
		if err2 := doc2.DataTo(&friendship); err2 == nil {
			return &friendship, nil
		}
	}

	return nil, nil // No existing friendship
}
