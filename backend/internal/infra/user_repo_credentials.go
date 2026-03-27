package infra

import (
	"context"

	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"gorm.io/gorm"
)

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
