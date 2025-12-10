package services

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"time"

	"github.com/yourusername/rbd-service/internal/models"
	"github.com/yourusername/rbd-service/internal/repository"
	"github.com/yourusername/rbd-service/pkg/utils"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	userRepo *repository.UserRepository
}

func NewAuthService() *AuthService {
	return &AuthService{
		userRepo: repository.NewUserRepository(),
	}
}

// Register creates a new user account
func (s *AuthService) Register(ctx context.Context, req *models.RegisterRequest) (*models.AuthResponse, error) {
	// Validate username
	if err := utils.ValidateUsername(req.Username); err != nil {
		return nil, err
	}

	// Validate password
	if err := utils.ValidatePassword(req.Password); err != nil {
		return nil, err
	}

	// Check if username already exists
	existingUser, _ := s.userRepo.GetUserByUsername(ctx, req.Username)
	if existingUser != nil {
		return nil, errors.New("username already taken")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Generate user ID
	userID := generateUserID()

	// Create user
	user := &models.User{
		UserID:       userID,
		Username:     req.Username,
		PasswordHash: string(hashedPassword),
		CreatedAt:    time.Now(),
		MutedAll:     false,
	}

	if err := s.userRepo.CreateUser(ctx, user); err != nil {
		return nil, err
	}

	// Generate token (simple random token for now, can be JWT later)
	token := generateToken()

	// Store token in token store
	GetTokenStore().StoreToken(token, userID)

	return &models.AuthResponse{
		UserID:   userID,
		Username: req.Username,
		Token:    token,
	}, nil
}

// Login authenticates a user
func (s *AuthService) Login(ctx context.Context, req *models.LoginRequest) (*models.AuthResponse, error) {
	// Get user by username
	user, err := s.userRepo.GetUserByUsername(ctx, req.Username)
	if err != nil {
		return nil, errors.New("invalid username or password")
	}

	// Compare password
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("invalid username or password")
	}

	// Generate token
	token := generateToken()

	// Store token in token store
	GetTokenStore().StoreToken(token, user.UserID)

	return &models.AuthResponse{
		UserID:   user.UserID,
		Username: user.Username,
		Token:    token,
	}, nil
}

// UpdateFCMToken updates the user's FCM token
func (s *AuthService) UpdateFCMToken(ctx context.Context, userID, fcmToken string) error {
	if fcmToken == "" {
		return errors.New("fcm token cannot be empty")
	}
	return s.userRepo.UpdateFCMToken(ctx, userID, fcmToken)
}

// Logout invalidates a user's token
func (s *AuthService) Logout(token string) {
	GetTokenStore().DeleteToken(token)
}

// Helper functions

func generateUserID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}
