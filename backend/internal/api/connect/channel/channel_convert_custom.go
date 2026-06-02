package channelconnect

import (
	"encoding/json"

	"github.com/anthropics/agentsmesh/backend/internal/domain/agentpod"
	"github.com/anthropics/agentsmesh/backend/internal/domain/channel"
	"github.com/anthropics/agentsmesh/backend/internal/domain/user"
	"github.com/anthropics/agentsmesh/backend/pkg/protoconv"
	channelv1 "github.com/anthropics/agentsmesh/proto/gen/go/channel/v1"
)

// contentJSONToProto encodes the rich AST as JSON. Field_custom helper
// referenced by ChannelMessage.content_json.
func contentJSONToProto(c *channel.MessageContent) *string {
	if c == nil {
		return nil
	}
	if b, err := json.Marshal(c); err == nil {
		s := string(b)
		return &s
	}
	return nil
}

// mentionsJSONToProto always marshals (even on zero value) so the wire field
// is `{}` rather than absent — matches the REST envelope.
func mentionsJSONToProto(m channel.MessageMentions) *string {
	if b, err := json.Marshal(m); err == nil {
		s := string(b)
		return &s
	}
	return nil
}

// senderUserToProto projects the preloaded user.User row into the wire shape.
func senderUserToProto(u *user.User) *channelv1.ChannelMessageSenderUser {
	if u == nil {
		return nil
	}
	return &channelv1.ChannelMessageSenderUser{
		Id:        u.ID,
		Username:  u.Username,
		Name:      protoconv.StringPtr(u.Name),
		AvatarUrl: protoconv.StringPtr(u.AvatarURL),
	}
}

// senderPodToProto projects the preloaded agentpod.Pod row into the wire shape.
// Agent is preloaded by channelRepository (Preload("SenderPodInfo.Agent")) so the
// client can render "ClaudeCode (abc12345)" when no alias is set.
func senderPodToProto(p *agentpod.Pod) *channelv1.ChannelMessageSenderPod {
	if p == nil {
		return nil
	}
	out := &channelv1.ChannelMessageSenderPod{
		PodKey: p.PodKey,
		Alias:  protoconv.StringPtr(p.Alias),
	}
	if p.Agent != nil {
		out.Agent = &channelv1.ChannelMessageSenderAgent{Name: p.Agent.Name}
	}
	return out
}

func toProtoChannelPod(p *agentpod.Pod) *channelv1.ChannelPod {
	if p == nil {
		return nil
	}
	return &channelv1.ChannelPod{
		Id:          p.ID,
		PodKey:      p.PodKey,
		Status:      p.Status,
		AgentStatus: p.AgentStatus,
		Alias:       protoconv.StringPtr(p.Alias),
	}
}

// optionalString returns *p as an unaliased copy when non-nil. Matches the
// pattern used by REST handlers for converting *string PATCH inputs.
func optionalString(p *string) *string {
	return protoconv.StringPtr(p)
}

// defaultListLimit / defaultListOffset preserve REST behavior — list
// channels paginates at 50 with offset 0. Conventions §5: explicit zero
// is preserved (only absent or non-positive defaults).
func defaultListLimit(p *int32, fallback int32) int32 {
	if p == nil || *p <= 0 {
		return fallback
	}
	return *p
}

func defaultListOffset(p *int32) int32 {
	if p == nil || *p < 0 {
		return 0
	}
	return *p
}
