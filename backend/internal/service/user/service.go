package user

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"github.com/anthropics/agentsmesh/backend/pkg/crypto"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound            = errors.New("user not found")
	ErrEmailAlreadyExists      = errors.New("email already exists")
	ErrUsernameExists          = errors.New("username already exists")
	ErrInvalidCredentials      = errors.New("invalid credentials")
	ErrUserInactive            = errors.New("user is inactive")
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

// Authenticate authenticates a user by email and password
func (s *Service) Authenticate(ctx context.Context, email, password string) (*user.User, error) {
	u, err := s.GetByEmail(ctx, email)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	if !u.IsActive {
		return nil, ErrUserInactive
	}

	if u.PasswordHash == nil || *u.PasswordHash == "" {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*u.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	// Update last login
	now := time.Now()
	s.db.WithContext(ctx).Model(u).Update("last_login_at", now)

	return u, nil
}

// GetOrCreateByOAuth gets or creates a user from OAuth identity
func (s *Service) GetOrCreateByOAuth(ctx context.Context, provider, providerUserID, providerUsername, email, name, avatarURL string) (*user.User, bool, error) {
	// Check if identity already exists
	var identity user.Identity
	if err := s.db.WithContext(ctx).Where("provider = ? AND provider_user_id = ?", provider, providerUserID).First(&identity).Error; err == nil {
		// Identity exists, get user
		u, err := s.GetByID(ctx, identity.UserID)
		return u, false, err
	}

	// Check if user with email exists
	var u *user.User
	var isNew bool
	existing, err := s.GetByEmail(ctx, email)
	if err == nil {
		u = existing
	} else {
		// Create new user
		username := providerUsername
		if username == "" {
			username = email
		}

		// Ensure username is unique
		for i := 0; i < 100; i++ {
			if _, err := s.GetByUsername(ctx, username); err != nil {
				break
			}
			username = providerUsername + "_" + string(rune('0'+i))
		}

		u = &user.User{
			Email:    email,
			Username: username,
			IsActive: true,
		}
		if name != "" {
			u.Name = &name
		}
		if avatarURL != "" {
			u.AvatarURL = &avatarURL
		}

		if err := s.db.WithContext(ctx).Create(u).Error; err != nil {
			return nil, false, err
		}
		isNew = true
	}

	// Create identity
	identity = user.Identity{
		UserID:         u.ID,
		Provider:       provider,
		ProviderUserID: providerUserID,
	}
	if providerUsername != "" {
		identity.ProviderUsername = &providerUsername
	}

	if err := s.db.WithContext(ctx).Create(&identity).Error; err != nil {
		return nil, false, err
	}

	return u, isNew, nil
}

// UpdateIdentityTokens updates OAuth tokens for an identity
// Tokens are encrypted using AES-GCM before storage
func (s *Service) UpdateIdentityTokens(ctx context.Context, userID int64, provider, accessToken, refreshToken string, expiresAt *time.Time) error {
	updates := map[string]interface{}{
		"token_expires_at": expiresAt,
	}

	// Encrypt tokens if encryption key is configured
	if s.encryptionKey != "" {
		if accessToken != "" {
			encrypted, err := crypto.EncryptWithKey(accessToken, s.encryptionKey)
			if err != nil {
				return err
			}
			updates["access_token_encrypted"] = encrypted
		}
		if refreshToken != "" {
			encrypted, err := crypto.EncryptWithKey(refreshToken, s.encryptionKey)
			if err != nil {
				return err
			}
			updates["refresh_token_encrypted"] = encrypted
		}
	} else {
		// Fallback: store as-is (not recommended for production)
		if accessToken != "" {
			updates["access_token_encrypted"] = accessToken
		}
		if refreshToken != "" {
			updates["refresh_token_encrypted"] = refreshToken
		}
	}

	return s.db.WithContext(ctx).Model(&user.Identity{}).
		Where("user_id = ? AND provider = ?", userID, provider).
		Updates(updates).Error
}

// GetIdentity returns an OAuth identity
func (s *Service) GetIdentity(ctx context.Context, userID int64, provider string) (*user.Identity, error) {
	var identity user.Identity
	if err := s.db.WithContext(ctx).Where("user_id = ? AND provider = ?", userID, provider).First(&identity).Error; err != nil {
		return nil, err
	}
	return &identity, nil
}

// GetIdentityByProvider returns an OAuth identity by provider (alias for GetIdentity)
func (s *Service) GetIdentityByProvider(ctx context.Context, userID int64, provider string) (*user.Identity, error) {
	return s.GetIdentity(ctx, userID, provider)
}

// DecryptedTokens holds decrypted OAuth tokens
type DecryptedTokens struct {
	AccessToken  string
	RefreshToken string
	ExpiresAt    *time.Time
}

// GetDecryptedTokens retrieves and decrypts OAuth tokens for an identity
func (s *Service) GetDecryptedTokens(ctx context.Context, userID int64, provider string) (*DecryptedTokens, error) {
	identity, err := s.GetIdentity(ctx, userID, provider)
	if err != nil {
		return nil, err
	}

	tokens := &DecryptedTokens{
		ExpiresAt: identity.TokenExpiresAt,
	}

	// Decrypt tokens if encryption key is configured
	if s.encryptionKey != "" {
		if identity.AccessTokenEncrypted != nil && *identity.AccessTokenEncrypted != "" {
			decrypted, err := crypto.DecryptWithKey(*identity.AccessTokenEncrypted, s.encryptionKey)
			if err != nil {
				return nil, err
			}
			tokens.AccessToken = decrypted
		}
		if identity.RefreshTokenEncrypted != nil && *identity.RefreshTokenEncrypted != "" {
			decrypted, err := crypto.DecryptWithKey(*identity.RefreshTokenEncrypted, s.encryptionKey)
			if err != nil {
				return nil, err
			}
			tokens.RefreshToken = decrypted
		}
	} else {
		// No encryption key - return as-is
		if identity.AccessTokenEncrypted != nil {
			tokens.AccessToken = *identity.AccessTokenEncrypted
		}
		if identity.RefreshTokenEncrypted != nil {
			tokens.RefreshToken = *identity.RefreshTokenEncrypted
		}
	}

	return tokens, nil
}

// ListIdentities returns all identities for a user
func (s *Service) ListIdentities(ctx context.Context, userID int64) ([]*user.Identity, error) {
	var identities []*user.Identity
	err := s.db.WithContext(ctx).Where("user_id = ?", userID).Find(&identities).Error
	return identities, err
}

// DeleteIdentity deletes an OAuth identity
func (s *Service) DeleteIdentity(ctx context.Context, userID int64, provider string) error {
	return s.db.WithContext(ctx).Where("user_id = ? AND provider = ?", userID, provider).Delete(&user.Identity{}).Error
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

// SetEmailVerificationToken generates and sets a verification token for the user
// Returns the token to be sent via email
func (s *Service) SetEmailVerificationToken(ctx context.Context, userID int64) (string, error) {
	token, err := generateToken()
	if err != nil {
		return "", err
	}

	expiresAt := time.Now().Add(24 * time.Hour)

	err = s.db.WithContext(ctx).Model(&user.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"email_verification_token":      token,
		"email_verification_expires_at": expiresAt,
	}).Error

	return token, err
}

// VerifyEmail verifies a user's email using the verification token
func (s *Service) VerifyEmail(ctx context.Context, token string) (*user.User, error) {
	var u user.User
	err := s.db.WithContext(ctx).
		Where("email_verification_token = ?", token).
		First(&u).Error
	if err != nil {
		return nil, ErrInvalidVerificationToken
	}

	// Check if token has expired
	if u.EmailVerificationExpiresAt == nil || time.Now().After(*u.EmailVerificationExpiresAt) {
		return nil, ErrInvalidVerificationToken
	}

	// Check if already verified
	if u.IsEmailVerified {
		return nil, ErrEmailAlreadyVerified
	}

	// Mark as verified and clear token
	err = s.db.WithContext(ctx).Model(&u).Updates(map[string]interface{}{
		"is_email_verified":             true,
		"email_verification_token":      nil,
		"email_verification_expires_at": nil,
	}).Error
	if err != nil {
		return nil, err
	}

	u.IsEmailVerified = true
	return &u, nil
}

// SetPasswordResetToken generates and sets a password reset token for the user
// Returns the token to be sent via email
func (s *Service) SetPasswordResetToken(ctx context.Context, email string) (string, *user.User, error) {
	u, err := s.GetByEmail(ctx, email)
	if err != nil {
		return "", nil, ErrUserNotFound
	}

	token, err := generateToken()
	if err != nil {
		return "", nil, err
	}

	expiresAt := time.Now().Add(1 * time.Hour)

	err = s.db.WithContext(ctx).Model(u).Updates(map[string]interface{}{
		"password_reset_token":      token,
		"password_reset_expires_at": expiresAt,
	}).Error

	return token, u, err
}

// ResetPassword resets the user's password using the reset token
func (s *Service) ResetPassword(ctx context.Context, token, newPassword string) (*user.User, error) {
	var u user.User
	err := s.db.WithContext(ctx).
		Where("password_reset_token = ?", token).
		First(&u).Error
	if err != nil {
		return nil, ErrInvalidResetToken
	}

	// Check if token has expired
	if u.PasswordResetExpiresAt == nil || time.Now().After(*u.PasswordResetExpiresAt) {
		return nil, ErrInvalidResetToken
	}

	// Hash new password
	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// Update password and clear reset token
	err = s.db.WithContext(ctx).Model(&u).Updates(map[string]interface{}{
		"password_hash":            string(hash),
		"password_reset_token":     nil,
		"password_reset_expires_at": nil,
	}).Error
	if err != nil {
		return nil, err
	}

	return &u, nil
}

// GetByVerificationToken returns a user by their verification token
func (s *Service) GetByVerificationToken(ctx context.Context, token string) (*user.User, error) {
	var u user.User
	if err := s.db.WithContext(ctx).Where("email_verification_token = ?", token).First(&u).Error; err != nil {
		return nil, ErrInvalidVerificationToken
	}
	return &u, nil
}
