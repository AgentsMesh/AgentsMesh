package infra

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"gorm.io/gorm"
)

// Compile-time interface check
var _ user.Repository = (*userRepo)(nil)

type userRepo struct {
	db *gorm.DB
}

// NewUserRepository creates a new GORM-based user repository
func NewUserRepository(db *gorm.DB) user.Repository {
	return &userRepo{db: db}
}

// --- User CRUD ---

func (r *userRepo) CreateUser(ctx context.Context, u *user.User) error {
	return r.db.WithContext(ctx).Create(u).Error
}

func (r *userRepo) GetByID(ctx context.Context, id int64) (*user.User, error) {
	var u user.User
	if err := r.db.WithContext(ctx).First(&u, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, user.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *userRepo) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	var u user.User
	if err := r.db.WithContext(ctx).Where("email = ?", email).First(&u).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, user.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *userRepo) GetByUsername(ctx context.Context, username string) (*user.User, error) {
	var u user.User
	if err := r.db.WithContext(ctx).Where("username = ?", username).First(&u).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, user.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *userRepo) EmailExists(ctx context.Context, email string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&user.User{}).Where("email = ?", email).Count(&count).Error
	return count > 0, err
}

func (r *userRepo) UsernameExists(ctx context.Context, username string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&user.User{}).Where("username = ?", username).Count(&count).Error
	return count > 0, err
}

func (r *userRepo) UpdateUser(ctx context.Context, id int64, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&user.User{}).Where("id = ?", id).Updates(updates).Error
}

func (r *userRepo) UpdateUserField(ctx context.Context, id int64, field string, value interface{}) error {
	return r.db.WithContext(ctx).Model(&user.User{}).Where("id = ?", id).Update(field, value).Error
}

func (r *userRepo) DeleteUser(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Delete(&user.User{}, id).Error
}

func (r *userRepo) SearchUsers(ctx context.Context, query string, limit int) ([]*user.User, error) {
	var users []*user.User
	err := r.db.WithContext(ctx).
		Where("username ILIKE ? OR name ILIKE ? OR email ILIKE ?", "%"+query+"%", "%"+query+"%", "%"+query+"%").
		Limit(limit).
		Find(&users).Error
	return users, err
}

// --- Auth token queries ---

func (r *userRepo) GetByVerificationToken(ctx context.Context, token string) (*user.User, error) {
	var u user.User
	if err := r.db.WithContext(ctx).Where("email_verification_token = ?", token).First(&u).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, user.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

func (r *userRepo) GetByResetToken(ctx context.Context, token string) (*user.User, error) {
	var u user.User
	if err := r.db.WithContext(ctx).Where("password_reset_token = ?", token).First(&u).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, user.ErrNotFound
		}
		return nil, err
	}
	return &u, nil
}

// --- Identity (OAuth) ---

func (r *userRepo) GetIdentityByProviderUser(ctx context.Context, provider, providerUserID string) (*user.Identity, error) {
	var identity user.Identity
	err := r.db.WithContext(ctx).
		Where("provider = ? AND provider_user_id = ?", provider, providerUserID).
		First(&identity).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, user.ErrIdentityNotFound
		}
		return nil, err
	}
	return &identity, nil
}

func (r *userRepo) GetIdentity(ctx context.Context, userID int64, provider string) (*user.Identity, error) {
	var identity user.Identity
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND provider = ?", userID, provider).
		First(&identity).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, user.ErrIdentityNotFound
		}
		return nil, err
	}
	return &identity, nil
}

func (r *userRepo) CreateIdentity(ctx context.Context, identity *user.Identity) error {
	return r.db.WithContext(ctx).Create(identity).Error
}

func (r *userRepo) UpdateIdentityFields(ctx context.Context, userID int64, provider string, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&user.Identity{}).
		Where("user_id = ? AND provider = ?", userID, provider).
		Updates(updates).Error
}

func (r *userRepo) ListIdentities(ctx context.Context, userID int64) ([]*user.Identity, error) {
	var identities []*user.Identity
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Find(&identities).Error
	return identities, err
}

func (r *userRepo) DeleteIdentity(ctx context.Context, userID int64, provider string) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND provider = ?", userID, provider).
		Delete(&user.Identity{}).Error
}

