package repository

import (
	"context"
	"errors"
	"strings"

	"cloud.google.com/go/firestore"
	"github.com/yourusername/rbd-service/internal/config"
	"github.com/yourusername/rbd-service/internal/models"
	"google.golang.org/api/iterator"
)

type UserRepository struct {
	client *firestore.Client
}

func NewUserRepository() *UserRepository {
	return &UserRepository{
		client: config.FirestoreClient,
	}
}

// CreateUser creates a new user in Firestore
func (r *UserRepository) CreateUser(ctx context.Context, user *models.User) error {
	_, err := r.client.Collection("users").Doc(user.UserID).Set(ctx, user)
	return err
}

// GetUserByID retrieves a user by their ID
func (r *UserRepository) GetUserByID(ctx context.Context, userID string) (*models.User, error) {
	doc, err := r.client.Collection("users").Doc(userID).Get(ctx)
	if err != nil {
		return nil, err
	}

	var user models.User
	if err := doc.DataTo(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

// GetUserByUsername retrieves a user by their username
func (r *UserRepository) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	iter := r.client.Collection("users").Where("username", "==", username).Limit(1).Documents(ctx)
	doc, err := iter.Next()
	if err == iterator.Done {
		return nil, errors.New("user not found")
	}
	if err != nil {
		return nil, err
	}

	var user models.User
	if err := doc.DataTo(&user); err != nil {
		return nil, err
	}

	return &user, nil
}

// UpdateFCMToken updates the user's FCM token
func (r *UserRepository) UpdateFCMToken(ctx context.Context, userID, fcmToken string) error {
	_, err := r.client.Collection("users").Doc(userID).Update(ctx, []firestore.Update{
		{Path: "fcmToken", Value: fcmToken},
	})
	return err
}

// UpdateMuteAll updates the user's mute all setting
func (r *UserRepository) UpdateMuteAll(ctx context.Context, userID string, mutedAll bool) error {
	_, err := r.client.Collection("users").Doc(userID).Update(ctx, []firestore.Update{
		{Path: "mutedAll", Value: mutedAll},
	})
	return err
}

// SearchUsersByUsername searches for users by username (case-insensitive prefix match)
func (r *UserRepository) SearchUsersByUsername(ctx context.Context, username string, limit int) ([]*models.User, error) {
	// Validate search query to prevent scanning entire collection
	if len(strings.TrimSpace(username)) < 2 {
		return []*models.User{}, nil // Require at least 2 characters
	}
	
	// Firestore doesn't support case-insensitive queries, so we'll get all users
	// and filter in Go (fine for small user base, for production use Algolia/Elasticsearch)
	iter := r.client.Collection("users").Documents(ctx)
	
	var users []*models.User
	searchLower := strings.ToLower(username)
	
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var user models.User
		if err := doc.DataTo(&user); err != nil {
			continue
		}

		// Case-insensitive prefix match
		usernameLower := strings.ToLower(user.Username)
		if strings.HasPrefix(usernameLower, searchLower) {
			users = append(users, &user)
			if len(users) >= limit {
				break
			}
		}
	}

	return users, nil
}
