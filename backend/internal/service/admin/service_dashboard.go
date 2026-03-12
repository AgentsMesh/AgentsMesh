package admin

import (
	"context"
	"fmt"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/organization"
	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
)

// DashboardStats represents dashboard statistics
type DashboardStats struct {
	TotalUsers          int64 `json:"total_users"`
	ActiveUsers         int64 `json:"active_users"`
	TotalOrganizations  int64 `json:"total_organizations"`
	TotalRunners        int64 `json:"total_runners"`
	OnlineRunners       int64 `json:"online_runners"`
	TotalPods           int64 `json:"total_pods"`
	ActivePods          int64 `json:"active_pods"`
	TotalSubscriptions  int64 `json:"total_subscriptions"`
	ActiveSubscriptions int64 `json:"active_subscriptions"`

	// Time-based metrics
	NewUsersToday     int64 `json:"new_users_today"`
	NewUsersThisWeek  int64 `json:"new_users_this_week"`
	NewUsersThisMonth int64 `json:"new_users_this_month"`
}

// GetDashboardStats retrieves dashboard statistics
func (s *Service) GetDashboardStats(ctx context.Context) (*DashboardStats, error) {
	stats := &DashboardStats{}

	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	weekAgo := today.AddDate(0, 0, -7)
	monthAgo := today.AddDate(0, -1, 0)

	// Total and active users
	if err := s.db.Model(&user.User{}).Count(&stats.TotalUsers); err != nil {
		return nil, fmt.Errorf("failed to count total users: %w", err)
	}
	if err := s.db.Model(&user.User{}).Where("is_active = ?", true).Count(&stats.ActiveUsers); err != nil {
		return nil, fmt.Errorf("failed to count active users: %w", err)
	}

	// Total organizations
	if err := s.db.Model(&organization.Organization{}).Count(&stats.TotalOrganizations); err != nil {
		return nil, fmt.Errorf("failed to count organizations: %w", err)
	}

	// Total and online runners
	if err := s.db.Table("runners").Count(&stats.TotalRunners); err != nil {
		return nil, fmt.Errorf("failed to count runners: %w", err)
	}
	if err := s.db.Table("runners").Where("status = ?", "online").Count(&stats.OnlineRunners); err != nil {
		return nil, fmt.Errorf("failed to count online runners: %w", err)
	}

	// Total and active pods
	if err := s.db.Table("pods").Count(&stats.TotalPods); err != nil {
		return nil, fmt.Errorf("failed to count pods: %w", err)
	}
	if err := s.db.Table("pods").Where("status IN ?", agentpod.ActiveStatuses()).Count(&stats.ActivePods); err != nil {
		return nil, fmt.Errorf("failed to count active pods: %w", err)
	}

	// Subscriptions
	if err := s.db.Table("subscriptions").Count(&stats.TotalSubscriptions); err != nil {
		return nil, fmt.Errorf("failed to count subscriptions: %w", err)
	}
	if err := s.db.Table("subscriptions").Where("status = ?", "active").Count(&stats.ActiveSubscriptions); err != nil {
		return nil, fmt.Errorf("failed to count active subscriptions: %w", err)
	}

	// New users today
	if err := s.db.Model(&user.User{}).Where("created_at >= ?", today).Count(&stats.NewUsersToday); err != nil {
		return nil, fmt.Errorf("failed to count new users today: %w", err)
	}

	// New users this week
	if err := s.db.Model(&user.User{}).Where("created_at >= ?", weekAgo).Count(&stats.NewUsersThisWeek); err != nil {
		return nil, fmt.Errorf("failed to count new users this week: %w", err)
	}

	// New users this month
	if err := s.db.Model(&user.User{}).Where("created_at >= ?", monthAgo).Count(&stats.NewUsersThisMonth); err != nil {
		return nil, fmt.Errorf("failed to count new users this month: %w", err)
	}

	return stats, nil
}
