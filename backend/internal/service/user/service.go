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

type Service struct {
	repo           user.Repository
	encryptionKey  string
	preDeleteHooks []func(ctx context.Context, userID int64) error
}

func NewService(repo user.Repository) *Service {
	return &Service{repo: repo}
}

func NewServiceWithEncryption(repo user.Repository, encryptionKey string) *Service {
	return &Service{
		repo:          repo,
		encryptionKey: encryptionKey,
	}
}

func (s *Service) SetEncryptionKey(key string) {
	s.encryptionKey = key
}

type CreateRequest struct {
	Email    string
	Username string
	Name     string
	Password string
}

func (s *Service) Create(ctx context.Context, req *CreateRequest) (*user.User, error) {
	exists, err := s.repo.EmailExists(ctx, req.Email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrEmailAlreadyExists
	}

	exists, err = s.repo.UsernameExists(ctx, req.Username)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, ErrUsernameExists
	}

	var passwordHash string
	if req.Password != "" {
		hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		if err != nil {
			slog.ErrorContext(ctx, "failed to hash password during user creation", "email", req.Email, "error", err)
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
		slog.ErrorContext(ctx, "failed to create user", "email", req.Email, "error", err)
		return nil, err
	}

	slog.InfoContext(ctx, "user created", "user_id", u.ID, "email", req.Email)
	return u, nil
}

func (s *Service) GetByID(ctx context.Context, id int64) (*user.User, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return u, nil
}

func (s *Service) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	u, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return u, nil
}

func (s *Service) GetByUsername(ctx context.Context, username string) (*user.User, error) {
	u, err := s.repo.GetByUsername(ctx, username)
	if err != nil {
		return nil, ErrUserNotFound
	}
	return u, nil
}

func (s *Service) Update(ctx context.Context, id int64, updates map[string]interface{}) (*user.User, error) {
	if err := s.repo.UpdateUser(ctx, id, updates); err != nil {
		return nil, err
	}
	return s.GetByID(ctx, id)
}

func (s *Service) UpdatePassword(ctx context.Context, id int64, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		slog.ErrorContext(ctx, "failed to hash password during update", "user_id", id, "error", err)
		return err
	}
	if err := s.repo.UpdateUserField(ctx, id, "password_hash", string(hash)); err != nil {
		slog.ErrorContext(ctx, "failed to update password", "user_id", id, "error", err)
		return err
	}
	slog.InfoContext(ctx, "password updated", "user_id", id)
	return nil
}

func (s *Service) AddPreDeleteHook(hook func(ctx context.Context, userID int64) error) {
	s.preDeleteHooks = append(s.preDeleteHooks, hook)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	for _, hook := range s.preDeleteHooks {
		if err := hook(ctx, id); err != nil {
			slog.ErrorContext(ctx, "pre-delete hook failed", "user_id", id, "error", err)
			return err
		}
	}
	if err := s.repo.DeleteUser(ctx, id); err != nil {
		slog.ErrorContext(ctx, "failed to delete user", "user_id", id, "error", err)
		return err
	}
	slog.InfoContext(ctx, "user deleted", "user_id", id)
	return nil
}

func (s *Service) Search(ctx context.Context, query string, limit int) ([]*user.User, error) {
	return s.repo.SearchUsers(ctx, query, limit)
}

func generateToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}
