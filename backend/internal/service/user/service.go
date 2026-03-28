package user

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"log/slog"

	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserNotFound             = errors.New("user not found")
	ErrEmailAlreadyExists       = errors.New("email already exists")
	ErrUsernameExists           = errors.New("username already exists")
	ErrInvalidCredentials       = errors.New("invalid credentials")
	ErrUserInactive             = errors.New("user is inactive")
	ErrInvalidVerificationToken = errors.New("invalid or expired verification token")
	ErrInvalidResetToken        = errors.New("invalid or expired reset token")
	ErrEmailAlreadyVerified     = errors.New("email already verified")
)

// Service handles user operations
type Service struct {
	repo           user.Repository
	encryptionKey  string
	preDeleteHooks []func(ctx context.Context, userID int64) error
}

// NewService creates a new user service
func NewService(repo user.Repository) *Service {
	return &Service{repo: repo}
}

// NewServiceWithEncryption creates a new user service with encryption support
func NewServiceWithEncryption(repo user.Repository, encryptionKey string) *Service {
	return &Service{
		repo:          repo,
		encryptionKey: encryptionKey,
	}
}

// SetEncryptionKey sets the encryption key for token encryption
func (s *Service) SetEncryptionKey(key string) {
	s.encryptionKey = key
}

// CreateRequest represents user creation request
type CreateRequest struct {
	Email    string
	Username string
	Name     string
	Password string
}

// Create creates a new user
func (s *Service) Create(ctx context.Context, req *CreateRequest) (*user.User, error) {
	// Check if email already exists
	exists, err := s.repo.EmailExists(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrEmailAlreadyExists
	}

	// Check if username already exists
	exists, err = s.repo.UsernameExists(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUsernameExists
	}

	// Hash password if provided
	var passwordHash string
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			slog.Error("failed to hash password during user creation", "email", req.Email, "error", err)
			return nil, err
		}
		passwordHash = string(hash)
	}

	u := &user.User{
		Email:    req.Email,
		Username: req.Username,
		IsActive: true,
	}
	if req.Name != "" {
		u.Name = &req.Name
	}
	if passwordHash != "" {
		u.PasswordHash = &passwordHash
	}

	if err := s.repo.CreateUser(ctx, u); err != nil {
		slog.Error("failed to create user", "email", req.Email, "error", err)
		return nil, err
	}

	slog.Info("user created", "user_id", u.ID, "email", req.Email)
	return u, nil
}

// GetByID returns a user by ID
func (s *Service) GetByID(ctx context.Context, id int64) (*user.User, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return u, nil
}

// GetByEmail returns a user by email
func (s *Service) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	u, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return u, nil
}

// GetByUsername returns a user by username
func (s *Service) GetByUsername(ctx context.Context, username string) (*user.User, error) {
	u, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return u, nil
}

// Update updates a user
func (s *Service) Update(ctx context.Context, id int64, updates map[string]interface{}) (*user.User, error) {
	if err := s.repo.UpdateUser(ctx, id, updates); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

// UpdatePassword updates a user's password
func (s *Service) UpdatePassword(ctx context.Context, id int64, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		slog.Error("failed to hash password during update", "user_id", id, "error", err)
		return err
	}
	if err := s.repo.UpdateUserField(ctx, id, "password_hash", string(hash)); err != nil {
		slog.Error("failed to update password", "user_id", id, "error", err)
		return err
	}
	slog.Info("password updated", "user_id", id)
	return nil
}

// AddPreDeleteHook registers a callback invoked before user deletion (for cleanup without FK CASCADE)
func (s *Service) AddPreDeleteHook(hook func(ctx context.Context, userID int64) error) {
	s.preDeleteHooks = append(s.preDeleteHooks, hook)
}

// Delete deletes a user after running all pre-delete cleanup hooks
func (s *Service) Delete(ctx context.Context, id int64) error {
	for _, hook := range s.preDeleteHooks {
		if err := hook(ctx, id); err != nil {
			slog.Error("pre-delete hook failed", "user_id", id, "error", err)
			return err
		}
	}
	if err := s.repo.DeleteUser(ctx, id); err != nil {
		slog.Error("failed to delete user", "user_id", id, "error", err)
		return err
	}
	slog.Info("user deleted", "user_id", id)
	return nil
}

// Search searches for users
func (s *Service) Search(ctx context.Context, query string, limit int) ([]*user.User, error) {
	return s.repo.SearchUsers(ctx, query, limit)
}

// generateToken generates a random token
func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
