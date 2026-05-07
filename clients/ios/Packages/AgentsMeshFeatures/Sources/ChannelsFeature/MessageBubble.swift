import SwiftUI
import AgentsMeshCore
import DesignSystem

/// 8 个 bubble variant — 设计稿 ios-channel-detail.pastel。
public enum MessageBubble {

    public struct Content<Inner: View>: View {
        let fillColor: Color
        let inner: Inner
        public init(fill: Color, @ViewBuilder content: () -> Inner) {
            self.fillColor = fill
            self.inner = content()
        }
        public var body: some View {
            HStack {
                inner
                    .frame(maxWidth: 268, alignment: .leading)
                    .padding(.horizontal, 14)
                    .padding(.vertical, 10)
                    .background(fillColor)
                    .overlay(RoundedRectangle(cornerRadius: 16).stroke(AMColors.border, lineWidth: 1))
                    .clipShape(RoundedRectangle(cornerRadius: 16))
                Spacer(minLength: 0)
            }
            .padding(.leading, 62)
            .padding(.trailing, 16)
            .padding(.vertical, 2)
            .accessibilityIdentifier("bubble-structured")
        }
    }

    public struct Text_: View {
        let content: String
        let fillColor: Color
        public init(_ content: String, fill: Color) {
            self.content = content
            self.fillColor = fill
        }
        public var body: some View {
            HStack {
                SwiftUI.Text(content)
                    .font(.system(size: 15))
                    .foregroundStyle(AMColors.foreground)
                    .lineLimit(nil)
                    .frame(maxWidth: 268, alignment: .leading)
                    .padding(.horizontal, 14)
                    .padding(.vertical, 10)
                    .background(fillColor)
                    .overlay(RoundedRectangle(cornerRadius: 16).stroke(AMColors.border, lineWidth: 1))
                    .clipShape(RoundedRectangle(cornerRadius: 16))
                Spacer(minLength: 0)
            }
            .padding(.leading, 62)
            .padding(.trailing, 16)
            .padding(.vertical, 2)
            .accessibilityIdentifier("bubble-text")
        }
    }

    public struct Code: View {
        let intro: String
        let code: String
        let fillColor: Color
        public init(intro: String, code: String, fill: Color) {
            self.intro = intro
            self.code = code
            self.fillColor = fill
        }
        public var body: some View {
            HStack {
                VStack(alignment: .leading, spacing: 8) {
                    SwiftUI.Text(intro)
                        .font(.system(size: 15))
                        .foregroundStyle(AMColors.foreground)
                    SwiftUI.Text(code)
                        .font(.system(size: 11, design: .monospaced))
                        .foregroundStyle(AMColors.codeText)
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .padding(10)
                        .background(AMColors.codeBg)
                        .clipShape(RoundedRectangle(cornerRadius: 8))
                }
                .frame(maxWidth: 268, alignment: .leading)
                .padding(.horizontal, 14)
                .padding(.vertical, 10)
                .background(fillColor)
                .overlay(RoundedRectangle(cornerRadius: 16).stroke(AMColors.border, lineWidth: 1))
                .clipShape(RoundedRectangle(cornerRadius: 16))
                Spacer(minLength: 0)
            }
            .padding(.leading, 62)
            .padding(.trailing, 16)
            .padding(.vertical, 2)
            .accessibilityIdentifier("bubble-code")
        }
    }

    public struct Quote: View {
        let quotedSender: String
        let quotedBody: String
        let reply: String
        public init(quotedSender: String, quotedBody: String, reply: String) {
            self.quotedSender = quotedSender
            self.quotedBody = quotedBody
            self.reply = reply
        }
        public var body: some View {
            HStack {
                VStack(alignment: .leading, spacing: 8) {
                    HStack(alignment: .top, spacing: 8) {
                        Rectangle().fill(AMColors.border).frame(width: 3)
                        VStack(alignment: .leading, spacing: 2) {
                            SwiftUI.Text(quotedSender)
                                .font(.system(size: 12, weight: .semibold))
                                .foregroundStyle(AMColors.mutedForeground)
                            SwiftUI.Text(quotedBody)
                                .font(.system(size: 13))
                                .foregroundStyle(AMColors.mutedForeground)
                                .lineLimit(2)
                        }
                    }
                    .padding(.leading, 4)
                    SwiftUI.Text(reply)
                        .font(.system(size: 15))
                        .foregroundStyle(AMColors.foreground)
                }
                .frame(maxWidth: 268, alignment: .leading)
                .padding(.horizontal, 14)
                .padding(.vertical, 10)
                .background(AMColors.card)
                .overlay(RoundedRectangle(cornerRadius: 16).stroke(AMColors.border, lineWidth: 1))
                .clipShape(RoundedRectangle(cornerRadius: 16))
                Spacer(minLength: 0)
            }
            .padding(.leading, 62)
            .padding(.trailing, 16)
            .padding(.vertical, 2)
            .accessibilityIdentifier("bubble-quote")
        }
    }

    public struct Mention: View {
        let prefix: String
        let mentionName: String
        let suffix: String
        public init(prefix: String, mention: String, suffix: String) {
            self.prefix = prefix
            self.mentionName = mention
            self.suffix = suffix
        }
        public var body: some View {
            HStack {
                (
                    SwiftUI.Text(prefix)
                        .foregroundColor(AMColors.foreground)
                    + SwiftUI.Text("@\(mentionName)")
                        .foregroundColor(AMColors.primary)
                        .fontWeight(.medium)
                    + SwiftUI.Text(suffix)
                        .foregroundColor(AMColors.foreground)
                )
                .font(.system(size: 15))
                .frame(maxWidth: 268, alignment: .leading)
                .padding(.horizontal, 14)
                .padding(.vertical, 10)
                .background(AMColors.card)
                .overlay(RoundedRectangle(cornerRadius: 16).stroke(AMColors.border, lineWidth: 1))
                .clipShape(RoundedRectangle(cornerRadius: 16))
                Spacer(minLength: 0)
            }
            .padding(.leading, 62)
            .padding(.trailing, 16)
            .padding(.vertical, 2)
            .accessibilityIdentifier("bubble-mention")
        }
    }

    public struct Link: View {
        let title: String
        let url: String
        public init(title: String, url: String) {
            self.title = title
            self.url = url
        }
        public var body: some View {
            HStack {
                VStack(alignment: .leading, spacing: 4) {
                    SwiftUI.Text(title)
                        .font(.system(size: 14, weight: .medium))
                        .foregroundStyle(AMColors.foreground)
                        .lineLimit(2)
                    HStack(spacing: 4) {
                        Image(systemName: "link")
                            .font(.system(size: 10))
                            .foregroundStyle(AMColors.primary)
                        SwiftUI.Text(url)
                            .font(.system(size: 12, design: .monospaced))
                            .foregroundStyle(AMColors.primary)
                            .lineLimit(1)
                    }
                }
                .frame(maxWidth: 268, alignment: .leading)
                .padding(14)
                .background(AMColors.card)
                .overlay(RoundedRectangle(cornerRadius: 16).stroke(AMColors.border, lineWidth: 1))
                .clipShape(RoundedRectangle(cornerRadius: 16))
                Spacer(minLength: 0)
            }
            .padding(.leading, 62)
            .padding(.trailing, 16)
            .padding(.vertical, 2)
            .accessibilityIdentifier("bubble-link")
        }
    }

    public struct Tool: View {
        let toolName: String
        let target: String
        public init(_ toolName: String, target: String) {
            self.toolName = toolName
            self.target = target
        }
        public var body: some View {
            HStack {
                VStack(alignment: .leading, spacing: 4) {
                    HStack(spacing: 6) {
                        SwiftUI.Text("🔧").font(.system(size: 13))
                        SwiftUI.Text(toolName)
                            .font(.system(size: 13, weight: .medium, design: .monospaced))
                            .foregroundStyle(AMColors.primaryStrong)
                    }
                    SwiftUI.Text(target)
                        .font(.system(size: 12, design: .monospaced))
                        .foregroundStyle(AMColors.mutedForeground)
                }
                .frame(maxWidth: 240, alignment: .leading)
                .padding(.horizontal, 14)
                .padding(.vertical, 10)
                .background(AMColors.card)
                .overlay(RoundedRectangle(cornerRadius: 16).stroke(AMColors.borderStrong, lineWidth: 1))
                .clipShape(RoundedRectangle(cornerRadius: 16))
                Spacer(minLength: 0)
            }
            .padding(.leading, 62)
            .padding(.trailing, 16)
            .padding(.vertical, 2)
            .accessibilityIdentifier("bubble-tool")
        }
    }

    public struct File: View {
        let filename: String
        let fileSize: String
        public init(_ filename: String, size: String) {
            self.filename = filename
            self.fileSize = size
        }
        public var body: some View {
            HStack {
                HStack(spacing: 10) {
                    VStack(spacing: 2) {
                        SwiftUI.Text("📄").font(.system(size: 18))
                        SwiftUI.Text("MD")
                            .font(.system(size: 8, weight: .semibold, design: .monospaced))
                            .foregroundStyle(AMColors.secondaryText)
                    }
                    .frame(width: 36, height: 44)
                    .background(AMColors.card)
                    .overlay(RoundedRectangle(cornerRadius: 6).stroke(AMColors.borderStrong, lineWidth: 1))
                    .clipShape(RoundedRectangle(cornerRadius: 6))
                    VStack(alignment: .leading, spacing: 2) {
                        SwiftUI.Text(filename).font(.system(size: 14, weight: .medium))
                            .foregroundStyle(AMColors.foreground).lineLimit(1)
                        SwiftUI.Text(fileSize).font(.system(size: 12)).foregroundStyle(AMColors.mutedForeground)
                    }
                }
                .frame(maxWidth: 244, alignment: .leading)
                .padding(12)
                .background(AMColors.card)
                .overlay(RoundedRectangle(cornerRadius: 16).stroke(AMColors.border, lineWidth: 1))
                .clipShape(RoundedRectangle(cornerRadius: 16))
                Spacer(minLength: 0)
            }
            .padding(.leading, 62)
            .padding(.trailing, 16)
            .padding(.vertical, 2)
            .accessibilityIdentifier("bubble-file")
        }
    }

    public struct Reactions: View {
        let entries: [(emoji: String, count: Int)]
        public init(_ entries: [(emoji: String, count: Int)]) { self.entries = entries }
        public var body: some View {
            HStack(spacing: 6) {
                ForEach(Array(entries.enumerated()), id: \.offset) { _, e in
                    HStack(spacing: 4) {
                        SwiftUI.Text(e.emoji).font(.system(size: 13))
                        SwiftUI.Text("\(e.count)")
                            .font(.system(size: 12, weight: .medium))
                            .foregroundStyle(AMColors.secondaryText)
                    }
                    .padding(.horizontal, 8)
                    .padding(.vertical, 2)
                    .background(AMColors.groupedBg)
                    .overlay(Capsule().stroke(AMColors.border, lineWidth: 1))
                    .clipShape(Capsule())
                }
                Spacer()
            }
            .padding(.leading, 62)
            .padding(.trailing, 16)
            .padding(.vertical, 4)
            .accessibilityIdentifier("bubble-reactions")
        }
    }

    public struct Typing: View {
        let name: String
        public init(_ name: String) { self.name = name }
        public var body: some View {
            HStack(spacing: 6) {
                SwiftUI.Text(name).font(.system(size: 12)).foregroundStyle(AMColors.mutedForeground)
                HStack(spacing: 3) {
                    ForEach(0..<3) { _ in
                        Circle().fill(AMColors.mutedForeground).frame(width: 5, height: 5)
                    }
                }
                Spacer()
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 4)
            .accessibilityIdentifier("bubble-typing")
        }
    }
}
