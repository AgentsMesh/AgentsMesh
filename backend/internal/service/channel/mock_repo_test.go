package channel

import (
	"context"
	"errors"
	"time"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
)

var errInjected = errors.New("injected error")

type errorInjectingRepo struct {
	channel.ChannelRepository
	failOn map[string]bool
}

func (r *errorInjectingRepo) shouldFail(method string) error {
	if r.failOn[method] {
		return errInjected
	}
	return nil
}

func (r *errorInjectingRepo) GetByID(ctx context.Context, id int64) (*channel.Channel, error) {
	if err := r.shouldFail("GetByID"); err != nil {
		return nil, err
	}
	return r.ChannelRepository.GetByID(ctx, id)
}

func (r *errorInjectingRepo) GetByOrgAndName(ctx context.Context, orgID int64, name string) (*channel.Channel, error) {
	if err := r.shouldFail("GetByOrgAndName"); err != nil {
		return nil, err
	}
	return r.ChannelRepository.GetByOrgAndName(ctx, orgID, name)
}

func (r *errorInjectingRepo) Create(ctx context.Context, ch *channel.Channel) error {
	if err := r.shouldFail("Create"); err != nil {
		return err
	}
	return r.ChannelRepository.Create(ctx, ch)
}

func (r *errorInjectingRepo) SetArchived(ctx context.Context, id int64, archived bool) error {
	if err := r.shouldFail("SetArchived"); err != nil {
		return err
	}
	return r.ChannelRepository.SetArchived(ctx, id, archived)
}

func (r *errorInjectingRepo) DeleteWithCleanup(ctx context.Context, id int64) error {
	if err := r.shouldFail("DeleteWithCleanup"); err != nil {
		return err
	}
	return r.ChannelRepository.DeleteWithCleanup(ctx, id)
}

func (r *errorInjectingRepo) DeleteChannelsByOrg(ctx context.Context, orgID int64) error {
	if err := r.shouldFail("DeleteChannelsByOrg"); err != nil {
		return err
	}
	return r.ChannelRepository.DeleteChannelsByOrg(ctx, orgID)
}

func (r *errorInjectingRepo) UpdateFields(ctx context.Context, id int64, updates map[string]interface{}) error {
	if err := r.shouldFail("UpdateFields"); err != nil {
		return err
	}
	return r.ChannelRepository.UpdateFields(ctx, id, updates)
}

func (r *errorInjectingRepo) IsMember(ctx context.Context, channelID, userID int64) (bool, error) {
	if err := r.shouldFail("IsMember"); err != nil {
		return false, err
	}
	return r.ChannelRepository.IsMember(ctx, channelID, userID)
}

func (r *errorInjectingRepo) GetMemberRole(ctx context.Context, channelID, userID int64) (string, error) {
	if err := r.shouldFail("GetMemberRole"); err != nil {
		return "", err
	}
	return r.ChannelRepository.GetMemberRole(ctx, channelID, userID)
}

func (r *errorInjectingRepo) AddMemberWithRole(ctx context.Context, channelID, userID int64, role string) error {
	if err := r.shouldFail("AddMemberWithRole"); err != nil {
		return err
	}
	return r.ChannelRepository.AddMemberWithRole(ctx, channelID, userID, role)
}

func (r *errorInjectingRepo) RemoveMember(ctx context.Context, channelID, userID int64) error {
	if err := r.shouldFail("RemoveMember"); err != nil {
		return err
	}
	return r.ChannelRepository.RemoveMember(ctx, channelID, userID)
}

func (r *errorInjectingRepo) SetMemberMuted(ctx context.Context, channelID, userID int64, muted bool) error {
	if err := r.shouldFail("SetMemberMuted"); err != nil {
		return err
	}
	return r.ChannelRepository.SetMemberMuted(ctx, channelID, userID, muted)
}

func (r *errorInjectingRepo) CreateMessage(ctx context.Context, msg *channel.Message) error {
	if err := r.shouldFail("CreateMessage"); err != nil {
		return err
	}
	return r.ChannelRepository.CreateMessage(ctx, msg)
}

func (r *errorInjectingRepo) GetMessages(ctx context.Context, channelID int64, before *time.Time, after *time.Time, limit int) ([]*channel.Message, error) {
	if err := r.shouldFail("GetMessages"); err != nil {
		return nil, err
	}
	return r.ChannelRepository.GetMessages(ctx, channelID, before, after, limit)
}

func (r *errorInjectingRepo) GetMessagesMentioning(ctx context.Context, channelID int64, podKey string, limit int) ([]*channel.Message, bool, error) {
	if err := r.shouldFail("GetMessagesMentioning"); err != nil {
		return nil, false, err
	}
	return r.ChannelRepository.GetMessagesMentioning(ctx, channelID, podKey, limit)
}

func (r *errorInjectingRepo) GetMessagesBefore(ctx context.Context, channelID int64, beforeID int64, limit int) ([]*channel.Message, error) {
	if err := r.shouldFail("GetMessagesBefore"); err != nil {
		return nil, err
	}
	return r.ChannelRepository.GetMessagesBefore(ctx, channelID, beforeID, limit)
}

func (r *errorInjectingRepo) GetRecentMessages(ctx context.Context, channelID int64, limit int) ([]*channel.Message, error) {
	if err := r.shouldFail("GetRecentMessages"); err != nil {
		return nil, err
	}
	return r.ChannelRepository.GetRecentMessages(ctx, channelID, limit)
}

func (r *errorInjectingRepo) GetMessageByID(ctx context.Context, msgID int64) (*channel.Message, error) {
	if err := r.shouldFail("GetMessageByID"); err != nil {
		return nil, err
	}
	return r.ChannelRepository.GetMessageByID(ctx, msgID)
}

func (r *errorInjectingRepo) UpdateMessage(ctx context.Context, msgID int64, body string, content *channel.MessageContent, mentions channel.MessageMentions) error {
	if err := r.shouldFail("UpdateMessage"); err != nil {
		return err
	}
	return r.ChannelRepository.UpdateMessage(ctx, msgID, body, content, mentions)
}

func (r *errorInjectingRepo) UpdateMessageMentions(ctx context.Context, msgID int64, mentions channel.MessageMentions) error {
	if err := r.shouldFail("UpdateMessageMentions"); err != nil {
		return err
	}
	return r.ChannelRepository.UpdateMessageMentions(ctx, msgID, mentions)
}

func (r *errorInjectingRepo) SoftDeleteMessage(ctx context.Context, msgID int64) error {
	if err := r.shouldFail("SoftDeleteMessage"); err != nil {
		return err
	}
	return r.ChannelRepository.SoftDeleteMessage(ctx, msgID)
}

func (r *errorInjectingRepo) UpsertAccess(ctx context.Context, channelID int64, podKey *string, userID *int64) error {
	if err := r.shouldFail("UpsertAccess"); err != nil {
		return err
	}
	return r.ChannelRepository.UpsertAccess(ctx, channelID, podKey, userID)
}

func (r *errorInjectingRepo) AddPodToChannel(ctx context.Context, channelID int64, podKey string) error {
	if err := r.shouldFail("AddPodToChannel"); err != nil {
		return err
	}
	return r.ChannelRepository.AddPodToChannel(ctx, channelID, podKey)
}

func (r *errorInjectingRepo) CreateBinding(ctx context.Context, binding *channel.PodBinding) error {
	if err := r.shouldFail("CreateBinding"); err != nil {
		return err
	}
	return r.ChannelRepository.CreateBinding(ctx, binding)
}

func (r *errorInjectingRepo) GetBindingByID(ctx context.Context, id int64) (*channel.PodBinding, error) {
	if err := r.shouldFail("GetBindingByID"); err != nil {
		return nil, err
	}
	return r.ChannelRepository.GetBindingByID(ctx, id)
}

func (r *errorInjectingRepo) GetBindingByPods(ctx context.Context, init, target string) (*channel.PodBinding, error) {
	if err := r.shouldFail("GetBindingByPods"); err != nil {
		return nil, err
	}
	return r.ChannelRepository.GetBindingByPods(ctx, init, target)
}

func (r *errorInjectingRepo) UpdateBindingFields(ctx context.Context, id int64, updates map[string]interface{}) error {
	if err := r.shouldFail("UpdateBindingFields"); err != nil {
		return err
	}
	return r.ChannelRepository.UpdateBindingFields(ctx, id, updates)
}

func (r *errorInjectingRepo) GetChannelPods(ctx context.Context, channelID int64) ([]*agentpod.Pod, error) {
	if err := r.shouldFail("GetChannelPods"); err != nil {
		return nil, err
	}
	return r.ChannelRepository.GetChannelPods(ctx, channelID)
}

func (r *errorInjectingRepo) GetMemberUserIDs(ctx context.Context, channelID int64) ([]int64, error) {
	if err := r.shouldFail("GetMemberUserIDs"); err != nil {
		return nil, err
	}
	return r.ChannelRepository.GetMemberUserIDs(ctx, channelID)
}

func (r *errorInjectingRepo) SearchMessages(ctx context.Context, channelID int64, query string, limit int) ([]*channel.Message, error) {
	if err := r.shouldFail("SearchMessages"); err != nil {
		return nil, err
	}
	return r.ChannelRepository.SearchMessages(ctx, channelID, query, limit)
}

func (r *errorInjectingRepo) SaveMessageEdit(ctx context.Context, edit *channel.MessageEdit) error {
	if err := r.shouldFail("SaveMessageEdit"); err != nil {
		return err
	}
	return r.ChannelRepository.SaveMessageEdit(ctx, edit)
}

func (r *errorInjectingRepo) GetMessageEdits(ctx context.Context, messageID int64) ([]*channel.MessageEdit, error) {
	if err := r.shouldFail("GetMessageEdits"); err != nil {
		return nil, err
	}
	return r.ChannelRepository.GetMessageEdits(ctx, messageID)
}
