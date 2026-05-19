package user

import (
	"context"
	"sync"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
)

type MockService struct {
	mu sync.RWMutex

	users      map[int64]*user.User
	identities map[int64][]*user.Identity
	nextID     int64

	CreateErr              error
	GetByIDErr             error
	GetByEmailErr          error
	GetByUsernameErr       error
	UpdateErr              error
	DeleteErr              error
	AuthenticateErr        error
	UpdatePasswordErr      error
	GetOrCreateByOAuthErr  error
	UpdateIdentityErr      error
	GetIdentityErr         error
	ListIdentitiesErr      error
	DeleteIdentityErr      error
	SearchErr              error

	CreatedUsers     []*CreateRequest
	UpdatedUsers     []map[string]interface{}
	DeletedUserIDs   []int64
	AuthAttempts     []authAttempt
	RecordLoginCalls []int64
	SearchQueries    []string
}

type authAttempt struct {
	Email    string
	Password string
}

func NewMockService() *MockService {
	return &MockService{
		users:      make(map[int64]*user.User),
		identities: make(map[int64][]*user.Identity),
		nextID:     1,
	}
}

func (m *MockService) Create(ctx context.Context, req *CreateRequest) (*user.User, error) {
	if m.CreateErr != nil {
		return nil, m.CreateErr
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.CreatedUsers = append(m.CreatedUsers, req)

	u := &user.User{
		ID:       m.nextID,
		Email:    req.Email,
		Username: req.Username,
		IsActive: true,
	}
	if req.Name != "" {
		u.Name = &req.Name
	}

	m.users[m.nextID] = u
	m.nextID++

	return u, nil
}

func (m *MockService) GetByID(ctx context.Context, id int64) (*user.User, error) {
	if m.GetByIDErr != nil {
		return nil, m.GetByIDErr
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if u, ok := m.users[id]; ok {
		return u, nil
	}
	return nil, ErrUserNotFound
}

func (m *MockService) GetByEmail(ctx context.Context, email string) (*user.User, error) {
	if m.GetByEmailErr != nil {
		return nil, m.GetByEmailErr
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, ErrUserNotFound
}

func (m *MockService) GetByUsername(ctx context.Context, username string) (*user.User, error) {
	if m.GetByUsernameErr != nil {
		return nil, m.GetByUsernameErr
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, u := range m.users {
		if u.Username == username {
			return u, nil
		}
	}
	return nil, ErrUserNotFound
}

func (m *MockService) Update(ctx context.Context, id int64, updates map[string]interface{}) (*user.User, error) {
	if m.UpdateErr != nil {
		return nil, m.UpdateErr
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.UpdatedUsers = append(m.UpdatedUsers, updates)

	u, ok := m.users[id]
	if !ok {
		return nil, ErrUserNotFound
	}

	if name, ok := updates["name"].(string); ok {
		u.Name = &name
	}
	if avatarURL, ok := updates["avatar_url"].(string); ok {
		u.AvatarURL = &avatarURL
	}

	return u, nil
}

func (m *MockService) UpdatePassword(ctx context.Context, id int64, password string) error {
	if m.UpdatePasswordErr != nil {
		return m.UpdatePasswordErr
	}
	return nil
}

func (m *MockService) Delete(ctx context.Context, id int64) error {
	if m.DeleteErr != nil {
		return m.DeleteErr
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.DeletedUserIDs = append(m.DeletedUserIDs, id)
	delete(m.users, id)
	return nil
}

func (m *MockService) Authenticate(ctx context.Context, email, password string) (*user.User, error) {
	m.mu.Lock()
	m.AuthAttempts = append(m.AuthAttempts, authAttempt{Email: email, Password: password})
	m.mu.Unlock()

	if m.AuthenticateErr != nil {
		return nil, m.AuthenticateErr
	}

	return m.GetByEmail(ctx, email)
}

func (m *MockService) RecordLogin(_ context.Context, userID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.RecordLoginCalls = append(m.RecordLoginCalls, userID)
}

func (m *MockService) GetOrCreateByOAuth(ctx context.Context, provider, providerUserID, providerUsername, email, name, avatarURL string) (*user.User, bool, error) {
	if m.GetOrCreateByOAuthErr != nil {
		return nil, false, m.GetOrCreateByOAuthErr
	}

	u, err := m.GetByEmail(ctx, email)
	if err == nil {
		return u, false, nil
	}

	u, err = m.Create(ctx, &CreateRequest{
		Email:    email,
		Username: providerUsername,
		Name:     name,
	})
	if err != nil {
		return nil, false, err
	}

	return u, true, nil
}

func (m *MockService) UpdateIdentityTokens(ctx context.Context, userID int64, provider, accessToken, refreshToken string, expiresAt *time.Time) error {
	if m.UpdateIdentityErr != nil {
		return m.UpdateIdentityErr
	}
	return nil
}

func (m *MockService) GetIdentity(ctx context.Context, userID int64, provider string) (*user.Identity, error) {
	if m.GetIdentityErr != nil {
		return nil, m.GetIdentityErr
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	identities := m.identities[userID]
	for _, id := range identities {
		if id.Provider == provider {
			return id, nil
		}
	}
	return nil, ErrUserNotFound
}

func (m *MockService) ListIdentities(ctx context.Context, userID int64) ([]*user.Identity, error) {
	if m.ListIdentitiesErr != nil {
		return nil, m.ListIdentitiesErr
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.identities[userID], nil
}

func (m *MockService) DeleteIdentity(ctx context.Context, userID int64, provider string) error {
	if m.DeleteIdentityErr != nil {
		return m.DeleteIdentityErr
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	identities := m.identities[userID]
	for i, id := range identities {
		if id.Provider == provider {
			m.identities[userID] = append(identities[:i], identities[i+1:]...)
			break
		}
	}
	return nil
}

func (m *MockService) Search(ctx context.Context, query string, limit int) ([]*user.User, error) {
	m.mu.Lock()
	m.SearchQueries = append(m.SearchQueries, query)
	m.mu.Unlock()

	if m.SearchErr != nil {
		return nil, m.SearchErr
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var results []*user.User
	for _, u := range m.users {
		results = append(results, u)
		if len(results) >= limit {
			break
		}
	}
	return results, nil
}

func (m *MockService) AddUser(u *user.User) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if u.ID == 0 {
		u.ID = m.nextID
		m.nextID++
	}
	m.users[u.ID] = u
}

func (m *MockService) AddIdentity(userID int64, identity *user.Identity) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.identities[userID] = append(m.identities[userID], identity)
}

// GetUsers returns all users (thread-safe).
func (m *MockService) GetUsers() []*user.User {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]*user.User, 0, len(m.users))
	for _, u := range m.users {
		result = append(result, u)
	}
	return result
}

func (m *MockService) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.users = make(map[int64]*user.User)
	m.identities = make(map[int64][]*user.Identity)
	m.nextID = 1
	m.CreatedUsers = nil
	m.UpdatedUsers = nil
	m.DeletedUserIDs = nil
	m.AuthAttempts = nil
	m.RecordLoginCalls = nil
	m.SearchQueries = nil
}

var _ Interface = (*MockService)(nil)
