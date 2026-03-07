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

// --- Git Credentials ---

func (r *userRepo) CreateGitCredential(ctx context.Context, credential *user.GitCredential) error {
	return r.db.WithContext(ctx).Create(credential).Error
}

func (r *userRepo) GetGitCredentialWithProvider(ctx context.Context, userID, credentialID int64) (*user.GitCredential, error) {
	var credential user.GitCredential
	err := r.db.WithContext(ctx).
		Preload("RepositoryProvider").
		Where("id = ? AND user_id = ?", credentialID, userID).
		First(&credential).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, user.ErrNotFound
		}
		return nil, err
	}
	return &credential, nil
}

func (r *userRepo) ListGitCredentialsWithProvider(ctx context.Context, userID int64) ([]*user.GitCredential, error) {
	var credentials []*user.GitCredential
	err := r.db.WithContext(ctx).
		Preload("RepositoryProvider").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&credentials).Error
	return credentials, err
}

func (r *userRepo) UpdateGitCredential(ctx context.Context, credential *user.GitCredential, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(credential).Updates(updates).Error
}

func (r *userRepo) DeleteGitCredential(ctx context.Context, userID, credentialID int64) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", credentialID, userID).
		Delete(&user.GitCredential{})
	return result.RowsAffected, result.Error
}

func (r *userRepo) GitCredentialNameExists(ctx context.Context, userID int64, name string, excludeID *int64) (bool, error) {
	query := r.db.WithContext(ctx).Model(&user.GitCredential{}).
		Where("user_id = ? AND name = ?", userID, name)
	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}
	var count int64
	err := query.Count(&count).Error
	return count > 0, err
}

func (r *userRepo) ClearUserDefaultCredential(ctx context.Context, userID, credentialID int64) error {
	return r.db.WithContext(ctx).
		Model(&user.User{}).
		Where("id = ? AND default_git_credential_id = ?", userID, credentialID).
		Update("default_git_credential_id", nil).Error
}

func (r *userRepo) SetDefaultGitCredential(ctx context.Context, userID, credentialID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Clear all defaults for this user
		if err := tx.Model(&user.GitCredential{}).
			Where("user_id = ?", userID).
			Update("is_default", false).Error; err != nil {
			return err
		}

		// Set the new default
		if err := tx.Model(&user.GitCredential{}).
			Where("id = ? AND user_id = ?", credentialID, userID).
			Update("is_default", true).Error; err != nil {
			return err
		}

		// Update user's default credential reference
		return tx.Model(&user.User{}).
			Where("id = ?", userID).
			Update("default_git_credential_id", credentialID).Error
	})
}

func (r *userRepo) ClearAllDefaultGitCredentials(ctx context.Context, userID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Clear all is_default flags
		if err := tx.Model(&user.GitCredential{}).
			Where("user_id = ?", userID).
			Update("is_default", false).Error; err != nil {
			return err
		}

		// Clear user's default credential reference
		return tx.Model(&user.User{}).
			Where("id = ?", userID).
			Update("default_git_credential_id", nil).Error
	})
}

func (r *userRepo) GetDefaultGitCredential(ctx context.Context, userID int64) (*user.GitCredential, error) {
	var credential user.GitCredential
	err := r.db.WithContext(ctx).
		Preload("RepositoryProvider").
		Where("user_id = ? AND is_default = ?", userID, true).
		First(&credential).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // No default set
		}
		return nil, err
	}
	return &credential, nil
}

// --- Repository Providers ---

func (r *userRepo) CreateRepositoryProvider(ctx context.Context, provider *user.RepositoryProvider) error {
	return r.db.WithContext(ctx).Create(provider).Error
}

func (r *userRepo) GetRepositoryProvider(ctx context.Context, userID, providerID int64) (*user.RepositoryProvider, error) {
	var provider user.RepositoryProvider
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", providerID, userID).
		First(&provider).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, user.ErrNotFound
		}
		return nil, err
	}
	return &provider, nil
}

func (r *userRepo) GetRepositoryProviderWithIdentity(ctx context.Context, userID, providerID int64) (*user.RepositoryProvider, error) {
	var provider user.RepositoryProvider
	err := r.db.WithContext(ctx).
		Preload("Identity").
		Where("id = ? AND user_id = ?", providerID, userID).
		First(&provider).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, user.ErrNotFound
		}
		return nil, err
	}
	return &provider, nil
}

func (r *userRepo) GetRepositoryProviderByTypeAndURL(ctx context.Context, userID int64, providerType, baseURL string) (*user.RepositoryProvider, error) {
	var provider user.RepositoryProvider
	err := r.db.WithContext(ctx).
		Preload("Identity").
		Where("user_id = ? AND provider_type = ? AND base_url = ? AND is_active = ?", userID, providerType, baseURL, true).
		First(&provider).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, user.ErrNotFound
		}
		return nil, err
	}
	return &provider, nil
}

func (r *userRepo) ListRepositoryProviders(ctx context.Context, userID int64) ([]*user.RepositoryProvider, error) {
	var providers []*user.RepositoryProvider
	err := r.db.WithContext(ctx).
		Preload("Identity").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&providers).Error
	return providers, err
}

func (r *userRepo) UpdateRepositoryProvider(ctx context.Context, provider *user.RepositoryProvider, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(provider).Updates(updates).Error
}

func (r *userRepo) DeleteRepositoryProvider(ctx context.Context, userID, providerID int64) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", providerID, userID).
		Delete(&user.RepositoryProvider{})
	return result.RowsAffected, result.Error
}

func (r *userRepo) RepositoryProviderNameExists(ctx context.Context, userID int64, name string, excludeID *int64) (bool, error) {
	query := r.db.WithContext(ctx).Model(&user.RepositoryProvider{}).
		Where("user_id = ? AND name = ?", userID, name)
	if excludeID != nil {
		query = query.Where("id != ?", *excludeID)
	}
	var count int64
	err := query.Count(&count).Error
	return count > 0, err
}

func (r *userRepo) GetRepositoryProviderByIdentityID(ctx context.Context, userID, identityID int64) (*user.RepositoryProvider, error) {
	var provider user.RepositoryProvider
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND identity_id = ?", userID, identityID).
		First(&provider).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, user.ErrNotFound
		}
		return nil, err
	}
	return &provider, nil
}

func (r *userRepo) SetDefaultRepositoryProvider(ctx context.Context, userID, providerID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Clear all defaults for this user
		if err := tx.Model(&user.RepositoryProvider{}).
			Where("user_id = ?", userID).
			Update("is_default", false).Error; err != nil {
			return err
		}

		// Set the new default
		return tx.Model(&user.RepositoryProvider{}).
			Where("id = ? AND user_id = ?", providerID, userID).
			Update("is_default", true).Error
	})
}
