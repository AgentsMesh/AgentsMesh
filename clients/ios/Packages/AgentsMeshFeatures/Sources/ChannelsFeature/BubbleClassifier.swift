import Foundation
import AgentsMeshCore

/// 把 ChannelMessageDto 派发到合适 bubble variant。优先级：
///   tool_call > file 附件 > 引用 (reply_to) > 结构化内容 > 代码块 > mention > link > text
/// 富结构(表格/列表/标题/多块)须先于 body 正则拦截：正则只认拍平后的 body，含
/// URL/@/``` 会被误判成单一气泡、丢失结构;仅单段落简单消息才走 body 正则。
enum BubbleVariant {
    case text
    case code(intro: String, code: String)
    case quote(sender: String, body: String, reply: String)
    case mention(prefix: String, name: String, suffix: String)
    case link(title: String, url: String)
    case tool(name: String, target: String)
    case file(filename: String, size: String)
}

enum BubbleClassifier {
    static func classify(_ msg: ChannelMessageDto) -> BubbleVariant {
        if msg.messageType == "tool_call" {
            let target = msg.body.split(separator: "\n", maxSplits: 1).dropFirst().first.map(String.init) ?? msg.body
            let name = msg.body.split(separator: "\n").first.map(String.init) ?? "tool"
            return .tool(name: name, target: target)
        }
        if let attachment = parseFileAttachment(msg.contentJson) {
            return .file(filename: attachment.name, size: attachment.size)
        }
        if msg.replyTo != nil, let quoted = parseQuote(msg.contentJson) {
            return .quote(sender: quoted.sender, body: quoted.body, reply: msg.body)
        }
        if isStructuredContent(msg.contentJson) {
            return .text
        }
        if let code = parseFencedCode(msg.body) {
            return .code(intro: code.intro, code: code.code)
        }
        if let mention = parseFirstMention(msg.body) {
            return .mention(prefix: mention.prefix, name: mention.name, suffix: mention.suffix)
        }
        if let link = parseFirstLink(msg.body) {
            return .link(title: link.title, url: link.url)
        }
        return .text
    }
}

private func isStructuredContent(_ json: String?) -> Bool {
    guard let blocks = MessageContentDecoder.decode(json)?.blocks, !blocks.isEmpty else {
        return false
    }
    if blocks.count == 1, blocks[0].type == "paragraph" {
        return false
    }
    return true
}

private func parseFencedCode(_ body: String) -> (intro: String, code: String)? {
    guard let start = body.range(of: "```") else { return nil }
    let after = body[start.upperBound...]
    guard let end = after.range(of: "```") else { return nil }
    let intro = String(body[..<start.lowerBound]).trimmingCharacters(in: .whitespacesAndNewlines)
    let code = String(after[..<end.lowerBound]).trimmingCharacters(in: .whitespacesAndNewlines)
    return (intro.isEmpty ? "Code:" : intro, code)
}

private func parseFirstMention(_ body: String) -> (prefix: String, name: String, suffix: String)? {
    guard let at = body.firstIndex(of: "@") else { return nil }
    let after = body[body.index(after: at)...]
    guard let end = after.firstIndex(where: { $0.isWhitespace || $0 == "," || $0 == "." }) ?? after.indices.last
    else { return nil }
    let name = String(after[..<end])
    guard !name.isEmpty else { return nil }
    let prefix = String(body[..<at])
    let suffix = String(body[end...])
    return (prefix, name, suffix)
}

private func parseFirstLink(_ body: String) -> (title: String, url: String)? {
    let pattern = "https?://[A-Za-z0-9._~:/?#\\[\\]@!$&'()*+,;=%-]+"
    guard let regex = try? NSRegularExpression(pattern: pattern) else { return nil }
    let range = NSRange(body.startIndex..<body.endIndex, in: body)
    guard let match = regex.firstMatch(in: body, range: range),
          let urlRange = Range(match.range, in: body) else { return nil }
    let url = String(body[urlRange])
    let title = body.replacingOccurrences(of: url, with: "")
        .trimmingCharacters(in: .whitespacesAndNewlines)
    return (title.isEmpty ? url : title, url)
}

private func parseFileAttachment(_ json: String?) -> (name: String, size: String)? {
    guard let json, let data = json.data(using: .utf8) else { return nil }
    if let dict = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
       let attachments = dict["attachments"] as? [[String: Any]],
       let first = attachments.first,
       let name = first["filename"] as? String {
        let size: String
        if let bytes = first["size"] as? Int { size = ByteCountFormatter().string(fromByteCount: Int64(bytes)) }
        else { size = "" }
        return (name, size)
    }
    return nil
}

private func parseQuote(_ json: String?) -> (sender: String, body: String)? {
    guard let json, let data = json.data(using: .utf8) else { return nil }
    if let dict = try? JSONSerialization.jsonObject(with: data) as? [String: Any],
       let quote = dict["quote"] as? [String: Any],
       let sender = quote["sender"] as? String,
       let body = quote["body"] as? String {
        return (sender, body)
    }
    return nil
}
