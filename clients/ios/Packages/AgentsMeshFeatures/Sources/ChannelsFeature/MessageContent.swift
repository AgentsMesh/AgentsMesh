import Foundation

// Swift mirror of the canonical channel-message AST defined in
// backend/internal/domain/channel/message_content.go and exposed to iOS via
// `ChannelMessageDto.contentJson`. Decoded once per message; the renderer
// (StructuredContentView) walks this tree.

public struct MessageContent: Codable, Equatable, Sendable {
    public var schemaVersion: Int?
    public var kind: String
    public var blocks: [Block]?
    public var attachmentKey: String?

    public enum CodingKeys: String, CodingKey {
        case schemaVersion = "schema_version"
        case kind
        case blocks
        case attachmentKey = "attachment_key"
    }
}

public struct Block: Codable, Equatable, Sendable {
    public var type: String
    public var elements: [InlineElement]?
    public var children: [Block]?
    public var level: Int?
    public var language: String?
    public var text: String?
    public var ordered: Bool?
    public var items: [[Block]]?
}

public struct InlineStyle: Codable, Equatable, Sendable {
    public var bold: Bool?
    public var italic: Bool?
    public var strike: Bool?
    public var code: Bool?
}

public struct InlineElement: Codable, Equatable, Sendable {
    public var type: String
    public var text: String?
    public var style: InlineStyle?
    public var entityType: String?
    public var entityKey: String?
    public var display: String?
    public var url: String?

    public enum CodingKeys: String, CodingKey {
        case type
        case text
        case style
        case entityType = "entity_type"
        case entityKey = "entity_key"
        case display
        case url
    }
}

public enum MessageContentDecoder {
    public static func decode(_ json: String?) -> MessageContent? {
        guard let s = json, let data = s.data(using: .utf8) else { return nil }
        return try? JSONDecoder().decode(MessageContent.self, from: data)
    }
}
