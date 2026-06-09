import SwiftUI
import DesignSystem

/// Renders a channel-message table block. Mirrors web's StructuredTable.tsx;
/// header cells use a semibold font, cells reuse StructuredInline for inline marks.
struct StructuredTableView: View {
    let block: Block

    var body: some View {
        if let rows = block.rows, !rows.isEmpty {
            ScrollView(.horizontal, showsIndicators: false) {
                Grid(alignment: .topLeading, horizontalSpacing: AMSpacing.s, verticalSpacing: AMSpacing.xs) {
                    ForEach(Array(rows.enumerated()), id: \.offset) { _, row in
                        GridRow {
                            ForEach(Array((row.cells ?? []).enumerated()), id: \.offset) { _, cell in
                                Text(StructuredInline.attributedString(from: cell.elements ?? []))
                                    .font(row.header == true ? AMTypography.bodySemibold : AMTypography.body)
                                    .foregroundStyle(AMColors.foreground)
                                    .multilineTextAlignment(alignment(cell.align))
                                    .gridCellAnchor(cellAnchor(cell.align))
                            }
                        }
                    }
                }
                .padding(AMSpacing.s)
                .overlay(
                    RoundedRectangle(cornerRadius: AMRadius.s, style: .continuous)
                        .stroke(AMColors.borderStrong, lineWidth: 1)
                )
            }
        } else {
            EmptyView()
        }
    }

    private func alignment(_ align: String?) -> TextAlignment {
        switch align {
        case "center": return .center
        case "right": return .trailing
        default: return .leading
        }
    }

    private func cellAnchor(_ align: String?) -> UnitPoint {
        switch align {
        case "center": return .center
        case "right": return .trailing
        default: return .leading
        }
    }
}
