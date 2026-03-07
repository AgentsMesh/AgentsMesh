package infra

import (
	"context"
	"errors"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"gorm.io/gorm"
)

// Compile-time interface compliance check.
var _ channel.ChannelRepository = (*channelRepository)(nil)

// channelAccess is the GORM model for the channel_access table.
type channelAccess struct {
	ID         int64     `gorm:"primaryKey"`
	ChannelID  int64     `gorm:"not null;index"`
	PodKey     *string   `gorm:"size:100;index"`
	UserID     *int64    `gorm:"index"`
	LastAccess time.Time `gorm:"not null;default:now()"`
}

func (channelAccess) TableName() string { return "channel_access" }

// channelPod is the GORM model for the channel_pods table.
type channelPod struct {
	ID        int64     `gorm:"primaryKey"`
	ChannelID int64     `gorm:"not null;index"`
	PodKey    string    `gorm:"size:100;not null"`
	JoinedAt  time.Time `gorm:"not null;default:now()"`
}

func (channelPod) TableName() string { return "channel_pods" }

type channelRepository struct {
	db *gorm.DB
}

// NewChannelRepository creates a new GORM-backed ChannelRepository.
func NewChannelRepository(db *gorm.DB) channel.ChannelRepository {
	return &channelRepository{db: db}
}

// --- Channel CRUD ---

func (r *channelRepository) GetByID(ctx context.Context, channelID int64) (*channel.Channel, error) {
	var ch channel.Channel
	if err := r.db.WithContext(ctx).First(&ch, channelID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &ch, nil
}

func (r *channelRepository) GetByOrgAndName(ctx context.Context, orgID int64, name string) (*channel.Channel, error) {
	var ch channel.Channel
	err := r.db.WithContext(ctx).
		Where("organization_id = ? AND name = ?", orgID, name).
		First(&ch).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &ch, nil
}

func (r *channelRepository) Create(ctx context.Context, ch *channel.Channel) error {
	return r.db.WithContext(ctx).Create(ch).Error
}

func (r *channelRepository) ListByOrg(ctx context.Context, orgID int64, filter *channel.ChannelListFilter) ([]*channel.Channel, int64, error) {
	query := r.db.WithContext(ctx).Model(&channel.Channel{}).Where("organization_id = ?", orgID)

	if !filter.IncludeArchived {
		query = query.Where("is_archived = ?", false)
	}
	if filter.RepositoryID != nil {
		query = query.Where("repository_id = ?", *filter.RepositoryID)
	}
	if filter.TicketID != nil {
		query = query.Where("ticket_id = ?", *filter.TicketID)
	}

	var total int64
	query.Count(&total)

	var channels []*channel.Channel
	if err := query.
		Order("updated_at DESC").
		Limit(filter.Limit).
		Offset(filter.Offset).
		Find(&channels).Error; err != nil {
		return nil, 0, err
	}
	return channels, total, nil
}

func (r *channelRepository) UpdateFields(ctx context.Context, channelID int64, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&channel.Channel{}).Where("id = ?", channelID).Updates(updates).Error
}

func (r *channelRepository) SetArchived(ctx context.Context, channelID int64, archived bool) error {
	return r.db.WithContext(ctx).Model(&channel.Channel{}).
		Where("id = ?", channelID).
		Update("is_archived", archived).Error
}

func (r *channelRepository) GetByTicketID(ctx context.Context, ticketID int64) ([]*channel.Channel, error) {
	var channels []*channel.Channel
	if err := r.db.WithContext(ctx).Where("ticket_id = ?", ticketID).Find(&channels).Error; err != nil {
		return nil, err
	}
	return channels, nil
}

// --- Messages ---

func (r *channelRepository) CreateMessage(ctx context.Context, msg *channel.Message) error {
	return r.db.WithContext(ctx).Create(msg).Error
}

func (r *channelRepository) TouchChannel(ctx context.Context, channelID int64) error {
	return r.db.WithContext(ctx).Model(&channel.Channel{}).
		Where("id = ?", channelID).
		Update("updated_at", time.Now()).Error
}

func (r *channelRepository) GetMessages(ctx context.Context, channelID int64, before *time.Time, limit int) ([]*channel.Message, error) {
	query := r.db.WithContext(ctx).Where("channel_id = ?", channelID)
	if before != nil {
		query = query.Where("created_at < ?", *before)
	}
	var messages []*channel.Message
	if err := query.
		Preload("SenderUser").
		Preload("SenderPodInfo").
		Preload("SenderPodInfo.AgentType").
		Order("created_at DESC").
		Limit(limit).
		Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

func (r *channelRepository) GetMessagesMentioning(ctx context.Context, channelID int64, pattern string, limit int) ([]*channel.Message, error) {
	var messages []*channel.Message
	if err := r.db.WithContext(ctx).
		Where("channel_id = ? AND content LIKE ?", channelID, "%"+pattern+"%").
		Order("created_at DESC").
		Limit(limit).
		Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

func (r *channelRepository) GetRecentMessages(ctx context.Context, channelID int64, limit int) ([]*channel.Message, error) {
	var messages []*channel.Message
	if err := r.db.WithContext(ctx).
		Where("channel_id = ?", channelID).
		Order("created_at DESC").
		Limit(limit).
		Find(&messages).Error; err != nil {
		return nil, err
	}
	return messages, nil
}

// --- Access tracking ---

func (r *channelRepository) UpsertAccess(ctx context.Context, channelID int64, podKey *string, userID *int64) error {
	query := r.db.WithContext(ctx).Where("channel_id = ?", channelID)
	if podKey != nil {
		query = query.Where("pod_key = ?", *podKey)
	}
	if userID != nil {
		query = query.Where("user_id = ?", *userID)
	}

	var existing channelAccess
	if err := query.First(&existing).Error; err == nil {
		return r.db.WithContext(ctx).Model(&existing).Update("last_access", time.Now()).Error
	}

	return r.db.WithContext(ctx).Create(&channelAccess{
		ChannelID:  channelID,
		PodKey:     podKey,
		UserID:     userID,
		LastAccess: time.Now(),
	}).Error
}

func (r *channelRepository) GetChannelsForPod(ctx context.Context, podKey string) ([]*channel.Channel, error) {
	var accesses []channelAccess
	if err := r.db.WithContext(ctx).Where("pod_key = ?", podKey).Find(&accesses).Error; err != nil {
		return nil, err
	}
	if len(accesses) == 0 {
		return []*channel.Channel{}, nil
	}

	ids := make([]int64, len(accesses))
	for i, a := range accesses {
		ids[i] = a.ChannelID
	}

	var channels []*channel.Channel
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&channels).Error; err != nil {
		return nil, err
	}
	return channels, nil
}

func (r *channelRepository) HasAccessed(ctx context.Context, channelID int64, podKey string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&channelAccess{}).
		Where("channel_id = ? AND pod_key = ?", channelID, podKey).
		Count(&count).Error
	return count > 0, err
}

func (r *channelRepository) GetAccessCount(ctx context.Context, channelID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&channelAccess{}).
		Where("channel_id = ?", channelID).
		Count(&count).Error
	return count, err
}

// --- Channel Pods ---

func (r *channelRepository) AddPodToChannel(ctx context.Context, channelID int64, podKey string) error {
	return r.db.WithContext(ctx).Create(&channelPod{
		ChannelID: channelID,
		PodKey:    podKey,
		JoinedAt:  time.Now(),
	}).Error
}

func (r *channelRepository) RemovePodFromChannel(ctx context.Context, channelID int64, podKey string) error {
	return r.db.WithContext(ctx).
		Where("channel_id = ? AND pod_key = ?", channelID, podKey).
		Delete(&channelPod{}).Error
}

func (r *channelRepository) GetChannelPods(ctx context.Context, channelID int64) ([]*agentpod.Pod, error) {
	var cps []channelPod
	if err := r.db.WithContext(ctx).Where("channel_id = ?", channelID).Find(&cps).Error; err != nil {
		return nil, err
	}
	if len(cps) == 0 {
		return []*agentpod.Pod{}, nil
	}

	keys := make([]string, len(cps))
	for i, cp := range cps {
		keys[i] = cp.PodKey
	}

	var pods []*agentpod.Pod
	if err := r.db.WithContext(ctx).Where("pod_key IN ?", keys).Find(&pods).Error; err != nil {
		return nil, err
	}
	return pods, nil
}

// --- Bindings (channel-level) ---

func (r *channelRepository) CreateBinding(ctx context.Context, binding *channel.PodBinding) error {
	return r.db.WithContext(ctx).Create(binding).Error
}

func (r *channelRepository) GetBindingByID(ctx context.Context, bindingID int64) (*channel.PodBinding, error) {
	var binding channel.PodBinding
	if err := r.db.WithContext(ctx).First(&binding, bindingID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &binding, nil
}

func (r *channelRepository) GetBindingByPods(ctx context.Context, initiator, target string) (*channel.PodBinding, error) {
	var binding channel.PodBinding
	err := r.db.WithContext(ctx).
		Where("initiator_pod = ? AND target_pod = ?", initiator, target).
		First(&binding).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &binding, nil
}

func (r *channelRepository) ListBindingsForPod(ctx context.Context, podKey string) ([]*channel.PodBinding, error) {
	var bindings []*channel.PodBinding
	if err := r.db.WithContext(ctx).
		Where("initiator_pod = ? OR target_pod = ?", podKey, podKey).
		Find(&bindings).Error; err != nil {
		return nil, err
	}
	return bindings, nil
}

func (r *channelRepository) UpdateBindingFields(ctx context.Context, bindingID int64, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&channel.PodBinding{}).
		Where("id = ?", bindingID).
		Updates(updates).Error
}
