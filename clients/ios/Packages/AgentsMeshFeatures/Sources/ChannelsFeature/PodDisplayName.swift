import AgentsMeshCore

/// Resolves a human-friendly display name for a channel-message pod sender.
/// Mirrors `clients/web/src/lib/pod-display-name.ts` — the iOS wire DTO carries
/// only alias + agent (no ticket/loop/title), so this is the trimmed subset.
public enum PodDisplayName {
    public static func of(_ info: SenderPodInfoDto, maxLength: Int = 20) -> String {
        if let alias = info.alias?.trimmed, !alias.isEmpty {
            return truncate(alias, max: maxLength)
        }
        let keyPrefix = shortKey(info.podKey)
        if let agent = info.agent?.name.trimmed, !agent.isEmpty {
            return "\(agent) (\(keyPrefix))"
        }
        return "\(keyPrefix)..."
    }

    /// Best-effort fallback when only the raw podKey string is available
    /// (e.g. legacy wire payloads with no senderPodInfo).
    public static func ofPodKey(_ podKey: String) -> String {
        return "\(shortKey(podKey))..."
    }

    public static func shortKey(_ podKey: String) -> String {
        return String(podKey.prefix(8))
    }

    private static func truncate(_ s: String, max: Int) -> String {
        if s.count > max {
            return String(s.prefix(max - 3)) + "..."
        }
        return s
    }
}

private extension String {
    var trimmed: String { trimmingCharacters(in: .whitespacesAndNewlines) }
}
