import Foundation
import SwiftUI
import DesignSystem

/// Helpers that convert a `[InlineElement]` sequence to an `AttributedString`
/// understood by SwiftUI's `Text`. Pure logic — keeps StructuredContentView's
/// SwiftUI body small.
public enum StructuredInline {
    public static func attributedString(from elements: [InlineElement]) -> AttributedString {
        var out = AttributedString()
        for el in elements {
            switch el.type {
            case "text":
                out.append(textRun(el))
            case "linebreak":
                out.append(AttributedString("\n"))
            case "mention":
                var run = AttributedString("@" + (el.display ?? el.entityKey ?? "unknown"))
                run.foregroundColor = AMColors.primary
                run.font = mentionFont(for: el.style)
                if el.style?.strike == true {
                    run.strikethroughStyle = .single
                }
                out.append(run)
            case "link":
                if let urlString = el.url, isAllowedScheme(urlString) {
                    var run = AttributedString(el.text ?? urlString)
                    run.link = URL(string: urlString)
                    run.foregroundColor = AMColors.primary
                    run.underlineStyle = .single
                    out.append(run)
                } else if let label = el.text {
                    out.append(AttributedString(label))
                }
            default:
                continue
            }
        }
        return out
    }

    private static func textRun(_ el: InlineElement) -> AttributedString {
        var run = AttributedString(el.text ?? "")
        let style = el.style
        var attrs = AttributeContainer()
        if style?.code == true {
            attrs.font = AMTypography.monoSm
            attrs.backgroundColor = AMColors.primarySoft
        } else if style?.bold == true && style?.italic == true {
            attrs.font = AMTypography.body.bold().italic()
        } else if style?.bold == true {
            attrs.font = AMTypography.bodySemibold
        } else if style?.italic == true {
            attrs.font = AMTypography.body.italic()
        }
        if style?.strike == true {
            attrs.strikethroughStyle = .single
        }
        run.mergeAttributes(attrs)
        return run
    }

    private static func isAllowedScheme(_ raw: String) -> Bool {
        guard let url = URL(string: raw), let scheme = url.scheme?.lowercased() else { return false }
        return scheme == "http" || scheme == "https" || scheme == "mailto"
    }

    private static func mentionFont(for style: InlineStyle?) -> Font {
        let bold = style?.bold == true
        let italic = style?.italic == true
        if bold && italic { return AMTypography.bodySemibold.italic() }
        if bold { return AMTypography.bodySemibold.bold() }
        if italic { return AMTypography.bodySemibold.italic() }
        return AMTypography.bodySemibold
    }
}
