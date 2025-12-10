package repository

import (
	"context"
	"time"

	"cloud.google.com/go/firestore"
	"github.com/yourusername/rbd-service/internal/config"
	"github.com/yourusername/rbd-service/internal/models"
	"google.golang.org/api/iterator"
)

type HistoryRepository struct {
	client *firestore.Client
}

func NewHistoryRepository() *HistoryRepository {
	return &HistoryRepository{
		client: config.FirestoreClient,
	}
}

// CreateHistory creates a new history record
func (r *HistoryRepository) CreateHistory(ctx context.Context, senderID, receiverID, senderUsername string) error {
	history := models.History{
		SenderID:       senderID,
		ReceiverID:     receiverID,
		SenderUsername: senderUsername,
		TriggeredAt:    time.Now(),
	}

	docRef, _, err := r.client.Collection("history").Add(ctx, history)
	if err != nil {
		return err
	}

	// Update with the generated ID
	_, err = docRef.Update(ctx, []firestore.Update{
		{Path: "historyId", Value: docRef.ID},
	})
	return err
}

// GetHistoryBetweenUsers retrieves history between two users
func (r *HistoryRepository) GetHistoryBetweenUsers(ctx context.Context, user1ID, user2ID string, page, limit int) ([]*models.History, int, error) {
	offset := (page - 1) * limit

	// Get all records where either user is sender or receiver
	var allHistory []*models.History

	// Query 1: user1 sent to user2
	iter1 := r.client.Collection("history").
		Where("senderId", "==", user1ID).
		Where("receiverId", "==", user2ID).
		OrderBy("triggeredAt", firestore.Desc).
		Documents(ctx)

	for {
		doc, err := iter1.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, 0, err
		}

		var history models.History
		if err := doc.DataTo(&history); err != nil {
			continue
		}
		allHistory = append(allHistory, &history)
	}

	// Query 2: user2 sent to user1
	iter2 := r.client.Collection("history").
		Where("senderId", "==", user2ID).
		Where("receiverId", "==", user1ID).
		OrderBy("triggeredAt", firestore.Desc).
		Documents(ctx)

	for {
		doc, err := iter2.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, 0, err
		}

		var history models.History
		if err := doc.DataTo(&history); err != nil {
			continue
		}
		allHistory = append(allHistory, &history)
	}

	// Sort by triggeredAt descending (most recent first)
	// Simple bubble sort for small datasets
	for i := 0; i < len(allHistory)-1; i++ {
		for j := 0; j < len(allHistory)-i-1; j++ {
			if allHistory[j].TriggeredAt.Before(allHistory[j+1].TriggeredAt) {
				allHistory[j], allHistory[j+1] = allHistory[j+1], allHistory[j]
			}
		}
	}

	total := len(allHistory)

	// Paginate
	start := offset
	end := offset + limit
	if start >= total {
		return []*models.History{}, total, nil
	}
	if end > total {
		end = total
	}

	return allHistory[start:end], total, nil
}

// GetLastTriggerTime gets the last time a user triggered another user
func (r *HistoryRepository) GetLastTriggerTime(ctx context.Context, senderID, receiverID string) (*time.Time, error) {
	iter := r.client.Collection("history").
		Where("senderId", "==", senderID).
		Where("receiverId", "==", receiverID).
		OrderBy("triggeredAt", firestore.Desc).
		Limit(1).
		Documents(ctx)

	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, nil // No history
	}
	if err != nil {
		return nil, err
	}

	var history models.History
	if err := doc.DataTo(&history); err != nil {
		return nil, err
	}

	return &history.TriggeredAt, nil
}
