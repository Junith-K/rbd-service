package repository

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/yourusername/rbd-service/internal/config"
	"github.com/yourusername/rbd-service/internal/models"
	"google.golang.org/api/iterator"
)

type CooldownRepository struct {
	client *firestore.Client
}

func NewCooldownRepository() *CooldownRepository {
	return &CooldownRepository{
		client: config.FirestoreClient,
	}
}

// CreateCooldown creates a new cooldown with specified duration in minutes
func (r *CooldownRepository) CreateCooldown(ctx context.Context, userID, targetUserID string, cooldownMinutes int) error {
	now := time.Now()
	expiresAt := now.Add(time.Duration(cooldownMinutes) * time.Minute)

	cooldown := models.Cooldown{
		UserID:       userID,
		TargetUserID: targetUserID,
		TriggeredAt:  now,
		ExpiresAt:    expiresAt,
	}

	docRef, _, err := r.client.Collection("cooldowns").Add(ctx, cooldown)
	if err != nil {
		return err
	}

	// Update with the generated ID
	_, err = docRef.Update(ctx, []firestore.Update{
		{Path: "cooldownId", Value: docRef.ID},
	})
	return err
}

// CheckActiveCooldown checks if there's an active cooldown between user and target
func (r *CooldownRepository) CheckActiveCooldown(ctx context.Context, userID, targetUserID string) (*models.Cooldown, error) {
	now := time.Now()

	iter := r.client.Collection("cooldowns").
		Where("userId", "==", userID).
		Where("targetUserId", "==", targetUserID).
		Where("expiresAt", ">", now).
		OrderBy("expiresAt", firestore.Desc).
		Limit(1).
		Documents(ctx)

	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, nil // No active cooldown
	}
	if err != nil {
		return nil, err
	}

	var cooldown models.Cooldown
	if err := doc.DataTo(&cooldown); err != nil {
		return nil, err
	}

	return &cooldown, nil
}

// CleanupExpiredCooldowns removes expired cooldowns (optional cleanup)
func (r *CooldownRepository) CleanupExpiredCooldowns(ctx context.Context) error {
	now := time.Now()

	iter := r.client.Collection("cooldowns").
		Where("expiresAt", "<", now).
		Documents(ctx)

	batch := r.client.Batch()
	count := 0

	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return err
		}

		batch.Delete(doc.Ref)
		count++

		// Firestore batch limit is 500
		if count >= 500 {
			if _, err := batch.Commit(ctx); err != nil {
				return err
			}
			batch = r.client.Batch()
			count = 0
		}
	}

	if count > 0 {
		_, err := batch.Commit(ctx)
		return err
	}

	return nil
}

// UpdateActiveCooldown updates an active cooldown's expiry time based on new cooldown duration
// Returns true if an active cooldown was updated, false if none exists
func (r *CooldownRepository) UpdateActiveCooldown(ctx context.Context, userID, targetUserID string, newCooldownMinutes int) (bool, error) {
	// Find active cooldown
	cooldown, err := r.CheckActiveCooldown(ctx, userID, targetUserID)
	if err != nil {
		return false, err
	}
	if cooldown == nil {
		return false, nil // No active cooldown to update
	}

	// Calculate new expiry time from the original trigger time
	newExpiresAt := cooldown.TriggeredAt.Add(time.Duration(newCooldownMinutes) * time.Minute)

	// Find the cooldown document
	iter := r.client.Collection("cooldowns").
		Where("userId", "==", userID).
		Where("targetUserId", "==", targetUserID).
		Where("cooldownId", "==", cooldown.CooldownID).
		Limit(1).
		Documents(ctx)

	doc, err := iter.Next()
	if err != nil {
		return false, err
	}

	// Update expiry time
	_, err = doc.Ref.Update(ctx, []firestore.Update{
		{Path: "expiresAt", Value: newExpiresAt},
	})
	if err != nil {
		return false, err
	}

	return true, nil
}
