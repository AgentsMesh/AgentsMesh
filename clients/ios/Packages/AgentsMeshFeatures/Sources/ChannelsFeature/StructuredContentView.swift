import SwiftUI
import DesignSystem

/// Renders the canonical channel-message AST. Mirrors web's
/// StructuredContent.tsx; consumes the same JSONB shape served by the backend.
public struct StructuredContentView: View {
    public let content: MessageContent

    public init(content: MessageContent) { self.content = content }

    public var body: some View {
        if let blocks = content.blocks, !blocks.isEmpty {
            VStack(alignment: .leading, spacing: AMSpacing.xs) {
                ForEach(Array(blocks.enumerated()), id: \.offset) { _, block in
                    BlockView(block: block)
                }
            }
        } else {
            EmptyView()
        }
    }
}

struct BlockView: View {
    let block: Block

    var body: some View {
        switch block.type {
        case "paragraph":
            paragraphText
        case "heading":
            headingText
        case "code_block":
            codeBlock
        case "quote":
            quoteBlock
        case "list":
            listBlock
        case "table":
            StructuredTableView(block: block)
        default:
            EmptyView()
        }
    }

    @ViewBuilder
    private var paragraphText: some View {
        if let inline = block.elements, !inline.isEmpty {
            Text(StructuredInline.attributedString(from: inline))
                .font(AMTypography.body)
                .foregroundStyle(AMColors.foreground)
                .textSelection(.enabled)
        }
    }

    @ViewBuilder
    private var headingText: some View {
        let lvl = max(1, min(block.level ?? 1, 3))
        let font: Font = lvl == 1 ? AMTypography.title2 : (lvl == 2 ? AMTypography.heading : AMTypography.bodySemibold)
        Text(StructuredInline.attributedString(from: block.elements ?? []))
            .font(font)
            .foregroundStyle(AMColors.foreground)
    }

    @ViewBuilder
    private var codeBlock: some View {
        Text(block.text ?? "")
            .font(AMTypography.mono)
            .foregroundStyle(AMColors.codeText)
            .padding(AMSpacing.s)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(AMColors.codeBg)
            .clipShape(RoundedRectangle(cornerRadius: AMRadius.s, style: .continuous))
            .textSelection(.enabled)
    }

    @ViewBuilder
    private var quoteBlock: some View {
        HStack(alignment: .top, spacing: AMSpacing.s) {
            Rectangle()
                .fill(AMColors.borderStrong)
                .frame(width: 3)
            VStack(alignment: .leading, spacing: AMSpacing.xs) {
                ForEach(Array((block.children ?? []).enumerated()), id: \.offset) { _, child in
                    BlockView(block: child)
                }
            }
        }
        .padding(.vertical, AMSpacing.xxs)
    }

    @ViewBuilder
    private var listBlock: some View {
        VStack(alignment: .leading, spacing: AMSpacing.xs) {
            ForEach(Array((block.items ?? []).enumerated()), id: \.offset) { idx, item in
                HStack(alignment: .top, spacing: AMSpacing.s) {
                    Text(bullet(at: idx))
                        .font(AMTypography.body)
                        .foregroundStyle(AMColors.mutedForeground)
                    VStack(alignment: .leading, spacing: AMSpacing.xs) {
                        ForEach(Array(item.enumerated()), id: \.offset) { _, child in
                            BlockView(block: child)
                        }
                    }
                }
            }
        }
    }

    private func bullet(at idx: Int) -> String {
        if block.ordered == true { return "\(idx + 1)." }
        return "•"
    }
}
