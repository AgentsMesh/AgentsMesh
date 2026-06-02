import ComposableArchitecture
import SwiftUI
import AgentsMeshCore
import DesignSystem

/// Lark-style time-sorted messages list. Mirrors
/// `design/mobile/pages/ios-channels-list.pastel`.
public struct ChannelsListView: View {
    @Perception.Bindable var store: StoreOf<ChannelsListFeature>

    public init(store: StoreOf<ChannelsListFeature>) { self.store = store }

    public var body: some View {
        ZStack {
            AMColors.card.ignoresSafeArea()
            content
        }
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarLeading) {
                Button { store.send(.avatarTapped) } label: {
                    HStack(spacing: 8) {
                        AMAvatar("S", size: .sm, bg: AMColors.primary)
                        Text("Channels")
                            .font(AMTypography.navTitle)
                            .foregroundStyle(AMColors.foreground)
                    }
                }
                .accessibilityIdentifier("nav-avatar")
            }
            ToolbarItem(placement: .topBarTrailing) {
                HStack(spacing: 0) {
                    Button { store.send(.composeTapped) } label: {
                        Image(systemName: "magnifyingglass").font(.system(size: 17)).foregroundStyle(AMColors.primary)
                    }
                    .accessibilityIdentifier("nav-search")
                    Button { store.send(.composeTapped) } label: {
                        Image(systemName: "plus").font(.system(size: 17, weight: .medium)).foregroundStyle(AMColors.primary)
                    }
                    .accessibilityIdentifier("nav-compose")
                }
            }
        }
        .onAppear { store.send(.onAppear) }
        .refreshable { store.send(.refreshRequested) }
    }

    @ViewBuilder
    private var content: some View {
        if store.isLoading && store.channels.isEmpty {
            ProgressView().frame(maxWidth: .infinity, maxHeight: .infinity)
        } else if store.channels.isEmpty {
            empty.frame(maxWidth: .infinity, maxHeight: .infinity)
        } else {
            ScrollView {
                LazyVStack(spacing: 0) {
                    ForEach(store.channels, id: \.id) { ch in
                        Button { store.send(.channelTapped(ch.id)) } label: {
                            ChannelRow(
                                channel: ch,
                                unread: store.unreadCounts[ch.id] ?? 0,
                                lastMessage: store.lastMessageByChannel[ch.id]
                            )
                        }
                        .buttonStyle(.plain)
                    }
                }
                .padding(.bottom, 100)
            }
        }
    }

    private var empty: some View {
        VStack(spacing: 12) {
            Image(systemName: "bubble.left.and.bubble.right")
                .font(.system(size: 48))
                .foregroundStyle(AMColors.mutedForeground)
            Text("No channels yet")
                .font(AMTypography.heading)
                .foregroundStyle(AMColors.foreground)
            Text("Tap + to start a conversation.")
                .font(AMTypography.body)
                .foregroundStyle(AMColors.mutedForeground)
        }
    }
}

struct ChannelRow: View {
    let channel: ChannelDto
    let unread: Int64
    let lastMessage: ChannelMessageDto?

    var body: some View {
        HStack(spacing: 12) {
            ZStack(alignment: .topTrailing) {
                AMAvatar(
                    String(channel.name.prefix(1).uppercased()),
                    shape: .roundedSquare,
                    size: .lg,
                    bg: tintFor(channel.name)
                )
                if unread > 0 {
                    Text(unread > 99 ? "99+" : "\(unread)")
                        .font(.system(size: 11, weight: .semibold))
                        .foregroundStyle(.white)
                        .padding(.horizontal, 6)
                        .padding(.vertical, 2)
                        .background(AMColors.destructive)
                        .clipShape(Capsule())
                        .overlay(Capsule().stroke(AMColors.card, lineWidth: 1.5))
                        .offset(x: 6, y: -6)
                }
            }
            VStack(alignment: .leading, spacing: 4) {
                HStack {
                    Text(channel.name)
                        .font(.system(size: 16, weight: .semibold))
                        .foregroundStyle(AMColors.foreground)
                        .lineLimit(1)
                    Spacer()
                    Text(timestamp)
                        .font(.system(size: 12))
                        .foregroundStyle(AMColors.mutedForeground)
                }
                Text(preview)
                    .font(.system(size: 13))
                    .foregroundStyle(AMColors.mutedForeground)
                    .lineLimit(1)
            }
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .frame(height: 76)
        .background(AMColors.card)
    }

    /// {sender}: {body} 截断到 1 行；缺消息时回退到 channel.description；都没有就空。
    private var preview: String {
        if let m = lastMessage {
            let sender = senderName(of: m)
            return "\(sender): \(m.body)"
        }
        return channel.description ?? ""
    }

    private func senderName(of m: ChannelMessageDto) -> String {
        if let user = m.senderUser {
            return user.name ?? user.username
        }
        if let info = m.senderPodInfo {
            return PodDisplayName.of(info)
        }
        if let key = m.senderPod {
            return PodDisplayName.ofPodKey(key)
        }
        return "Unknown"
    }

    private var timestamp: String {
        guard let iso = lastMessage?.createdAt else { return "" }
        return relativeTime(from: iso)
    }

    private func tintFor(_ name: String) -> Color {
        let hash = abs(name.hashValue)
        let palette: [Color] = [.blue, AMColors.primary, AMColors.purple, AMColors.success, AMColors.warning]
        return palette[hash % palette.count]
    }
}

/// 仅依赖 Foundation 的 ISO8601 → 相对时间（"2m"、"3h"、"5d"）。
func relativeTime(from iso8601: String) -> String {
    let f = ISO8601DateFormatter()
    f.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
    let date = f.date(from: iso8601) ?? {
        let f2 = ISO8601DateFormatter()
        f2.formatOptions = [.withInternetDateTime]
        return f2.date(from: iso8601)
    }()
    guard let d = date else { return "" }
    let secs = max(0, Int(Date().timeIntervalSince(d)))
    if secs < 60 { return "just now" }
    if secs < 3600 { return "\(secs / 60)m" }
    if secs < 86_400 { return "\(secs / 3600)h" }
    return "\(secs / 86_400)d"
}
