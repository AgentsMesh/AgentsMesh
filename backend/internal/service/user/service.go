package user

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"

	"github.com/AgentsMesh/AgentsMesh/backend/internal/domain/user"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
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
	db            *gorm.DB
	encryptionKey string
}

// NewService creates a new user service
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// NewServiceWithEncryption creates a new user service with encryption support
func NewServiceWithEncryption(db *gorm.DB, encryptionKey string) *Service {
	return &Service{
		db:            db,
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
	var existing user.User
	if err := s.db.WithContext(ctx).Where("email = ?", req.Email).First(&existing).Error; err == nil {
		return nil, ErrEmailAlreadyExists
	}

	// Check if username already exists
	if err := s.db.WithContext(ctx).Where("username = ?", req.Username).First(&existing).Error; err == nil {
		return nil, ErrUsernameExists
	}

	// Hash password if provided
	var passwordHash string
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
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

	if err := s.db.WithContext(ctx).Create(u).Error; err != nil {
		return nil, err
	}

	return u, nil
}

// GetByID returns a user by ID
func (s *Service) GetByID(ctx context.Context, id int64) (*user.User, error) {
	var u user.User
	if err := s.db.WithContext(ctx).First(&u, id).Error; err != nil {
		return nil, ErrUserNotFound
	}
	return &u, nil
}

// GetByEmail returns a user by email
func (s *Service) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	var u user.User
	if err := s.db.WithContext(ctx).Where("email = ?", email).First(&u).Error; err != nil {
		return nil, ErrUserNotFound
	}
	return &u, nil
}

// GetByUsername returns a user by username
func (s *Service) GetByUsername(ctx context.Context, username string) (*user.User, error) {
	var u user.User
	if err := s.db.WithContext(ctx).Where("username = ?", username).First(&u).Error; err != nil {
		return nil, ErrUserNotFound
	}
	return &u, nil
}

// Update updates a user
func (s *Service) Update(ctx context.Context, id int64, updates map[string]interface{}) (*user.User, error) {
	if err := s.db.WithContext(ctx).Model(&user.User{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

// UpdatePassword updates a user's password
func (s *Service) UpdatePassword(ctx context.Context, id int64, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.db.WithContext(ctx).Model(&user.User{}).Where("id = ?", id).Update("password_hash", string(hash)).Error
}

// Delete deletes a user
func (s *Service) Delete(ctx context.Context, id int64) error {
	return s.db.WithContext(ctx).Delete(&user.User{}, id).Error
}

// Search searches for users
func (s *Service) Search(ctx context.Context, query string, limit int) ([]*user.User, error) {
	var users []*user.User
	err := s.db.WithContext(ctx).
		Where("username ILIKE ? OR name ILIKE ? OR email ILIKE ?", "%"+query+"%", "%"+query+"%", "%"+query+"%").
		Limit(limit).
		Find(&users).Error
	return users, err
}

// generateToken generates a random token
func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
