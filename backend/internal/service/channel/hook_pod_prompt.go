package channel

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	channelDomain "github.com/anthropics/agentsmesh/backend/internal/domain/channel"
)

// podMentionTextLen matches the frontend's mentionText = podKey.slice(0, 8)
const podMentionTextLen = 8

// PodPromptRouter sends prompts to a pod (mode-agnostic: PTY or ACP).
type PodPromptRouter interface {
	RoutePrompt(podKey string, prompt string) error
}

// SystemMessageWriter creates system messages directly (bypassing hooks to avoid recursion)
type SystemMessageWriter interface {
	CreateMessage(ctx context.Context, msg *channelDomain.Message) error
}

// NewPodPromptHook creates a hook that sends @mentioned pod content to their PTYs.
// When a pod is unreachable, it writes a system message to the channel so the user
// gets visible feedback instead of silent failure.
func NewPodPromptHook(router PodPromptRouter, msgWriter SystemMessageWriter) PostSendHook {
	return func(ctx context.Context, mc *MessageContext) error {
		if router == nil || mc.Mentions == nil || len(mc.Mentions.PodKeys) == 0 {
			return nil
		}

		prompt := buildPodPrompt(mc.Message.Content, mc.Channel.Name, mc.Channel.ID, mc.Mentions.PodKeys)

		for _, podKey := range mc.Mentions.PodKeys {
			// Skip if the message was sent by this pod (don't echo back)
			if mc.Message.SenderPod != nil && *mc.Message.SenderPod == podKey {
				continue
			}

			if err := router.RoutePrompt(podKey, prompt+"\r"); err != nil {
				slog.Warn("pod unreachable for prompt",
					"pod_key", podKey,
					"channel", mc.Channel.Name,
					"error", err,
				)
				// Write a system message so the user knows the pod didn't receive it
				writeOfflineNotice(ctx, msgWriter, mc.Message.ChannelID, podKey)
				continue
			}
		}

		return nil
	}
}

// writeOfflineNotice creates a system message indicating that a pod is offline.
// Uses the repo directly to bypass the PostSendHook pipeline (avoids recursion).
func writeOfflineNotice(ctx context.Context, w SystemMessageWriter, channelID int64, podKey string) {
	if w == nil {
		return
	}
	msg := &channelDomain.Message{
		ChannelID:   channelID,
		MessageType: channelDomain.MessageTypeSystem,
		Content:     fmt.Sprintf("@%s is offline and cannot receive this message.", podKey),
	}
	if err := w.CreateMessage(ctx, msg); err != nil {
		slog.Error("failed to write pod-offline system message", "error", err)
	}
}

// stripPodMentions removes @mention text for the given pod keys from the content.
// The frontend uses podKey[:8] as the mention text (see useMentionCandidates.ts).
func stripPodMentions(content string, podKeys []string) string {
	result := content
	for _, key := range podKeys {
		mention := key
		if len(mention) > podMentionTextLen {
			mention = mention[:podMentionTextLen]
		}
		// Strip "@mention " (with trailing space) first, then bare "@mention"
		result = strings.ReplaceAll(result, "@"+mention+" ", "")
		result = strings.ReplaceAll(result, "@"+mention, "")
	}
	return strings.TrimSpace(result)
}

// buildPodPrompt builds a context-aware prompt matching the frontend's buildChannelPrompt.
// Strips @pod mentions from content and wraps with channel context + reply instruction.
// Includes channel_id so agents without built-in skills (e.g. Codex) can use
// the send_channel_message MCP tool to reply directly.
func buildPodPrompt(content, channelName string, channelID int64, podKeys []string) string {
	rawPrompt := stripPodMentions(content, podKeys)
	return fmt.Sprintf("Message from channel(#%s, channel_id=%d): %s\n\nIf you finish it, please reply to this channel using send_channel_message(channel_id=%d).", channelName, channelID, rawPrompt, channelID)
}
