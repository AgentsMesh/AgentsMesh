import ComposableArchitecture
import SwiftUI
import AgentsMeshCore
import DesignSystem

/// Lark-style channel detail with bubble timeline + bottom composer.
/// Mirrors `design/mobile/pages/ios-channel-detail.pastel`.
public struct ChannelDetailView: View {
    @Perception.Bindable var store: StoreOf<ChannelDetailFeature>

    public init(store: StoreOf<ChannelDetailFeature>) { self.store = store }

    public var body: some View {
        ZStack {
            AMColors.groupedBg.ignoresSafeArea()
            VStack(spacing: 0) {
                messageList
                composer
            }
        }
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .principal) {
                VStack(spacing: 0) {
                    Text(store.channel?.name ?? "—")
                        .font(.system(size: 17, weight: .semibold))
                    if let count = store.channel?.memberCount {
                        Text("\(count) members")
                            .font(.system(size: 11))
                            .foregroundStyle(AMColors.mutedForeground)
                    }
                }
            }
        }
        .onAppear { store.send(.onAppear) }
    }

    @ViewBuilder
    private var messageList: some View {
        ScrollViewReader { proxy in
            ScrollView {
                LazyVStack(alignment: .leading, spacing: 0) {
                    Text(dateLabel)
                        .font(.system(size: 12, weight: .medium))
                        .foregroundStyle(AMColors.mutedForeground)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 12)

                    ForEach(store.messages, id: \.id) { msg in
                        bubbleFor(msg)
                            .id(msg.id)
                    }
                }
            }
            .onChange(of: store.messages.count) { _ in
                if let last = store.messages.last {
                    withAnimation { proxy.scrollTo(last.id, anchor: .bottom) }
                }
            }
        }
    }

    @ViewBuilder
    private func bubbleFor(_ msg: ChannelMessageDto) -> some View {
        VStack(alignment: .leading, spacing: 0) {
            senderHeader(msg)
            bubbleBody(msg)
        }
    }

    @ViewBuilder
    private func bubbleBody(_ msg: ChannelMessageDto) -> some View {
        switch BubbleClassifier.classify(msg) {
        case .code(let intro, let code):
            MessageBubble.Code(intro: intro, code: code, fill: AMColors.card)
        case .quote(let sender, let quoted, let reply):
            MessageBubble.Quote(quotedSender: sender, quotedBody: quoted, reply: reply)
        case .mention(let prefix, let mention, let suffix):
            MessageBubble.Mention(prefix: prefix, mention: mention, suffix: suffix)
        case .link(let title, let url):
            MessageBubble.Link(title: title, url: url)
        case .tool(let name, let target):
            MessageBubble.Tool(name, target: target)
        case .file(let filename, let size):
            MessageBubble.File(filename, size: size)
        case .text:
            if let ast = MessageContentDecoder.decode(msg.contentJson), (ast.blocks?.isEmpty == false) {
                MessageBubble.Content(fill: AMColors.card) {
                    StructuredContentView(content: ast)
                }
            } else {
                MessageBubble.Text_(msg.body, fill: AMColors.card)
            }
        }
    }

    @ViewBuilder
    private func senderHeader(_ msg: ChannelMessageDto) -> some View {
        let name = msg.senderUser?.name ?? msg.senderUser?.username ?? "User"
        let initial = String(name.prefix(1).uppercased())
        HStack(alignment: .top, spacing: 10) {
            AMAvatar(initial, size: .sm, bg: AMColors.primary)
            Text(name)
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(AMColors.foreground)
                .padding(.top, 8)
            Spacer()
        }
        .padding(.horizontal, 16)
        .padding(.top, 12)
        .padding(.bottom, 2)
    }

    private var composer: some View {
        ChannelComposer(
            placeholder: "Send to #\(store.channel?.name ?? "")",
            text: $store.draft,
            isSending: store.isSending,
            onSend: { store.send(.sendTapped) }
        )
    }

    private var dateLabel: String { "Today" }
}
