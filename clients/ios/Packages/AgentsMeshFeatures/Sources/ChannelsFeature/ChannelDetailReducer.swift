import ComposableArchitecture
import Foundation
import AgentsMeshCore
import CoreClient

@Reducer
public struct ChannelDetailFeature {
    public init() {}

    @ObservableState
    public struct State: Equatable {
        public let channelId: Int64
        public var channel: ChannelDto?
        public var messages: [ChannelMessageDto] = []
        public var draft: String = ""
        public var isLoading: Bool = false
        public var errorMessage: String?
        public var isSending: Bool = false

        public init(channelId: Int64) {
            self.channelId = channelId
        }
    }

    public enum Action: BindableAction, Sendable {
        case onAppear
        case channelLoaded(ChannelDto)
        case messagesLoaded([ChannelMessageDto])
        case loadFailed(String)
        case sendTapped
        case messageSent(ChannelMessageDto)
        case sendFailed(String)
        case binding(BindingAction<State>)
    }

    @Dependency(\.coreClient) var core

    public var body: some ReducerOf<Self> {
        BindingReducer()
        Reduce { state, action in
            switch action {
            case .onAppear:
                state.isLoading = true
                let id = state.channelId
                return .run { send in
                    do {
                        async let chan = core.getChannel(id)
                        async let msgs = core.getChannelMessages(id, 50, nil)
                        let (c, m) = try await (chan, msgs)
                        await send(.channelLoaded(c))
                        await send(.messagesLoaded(m.messages.reversed()))
                    } catch {
                        await send(.loadFailed(error.localizedDescription))
                    }
                }

            case .channelLoaded(let c):
                state.channel = c
                return .none

            case .messagesLoaded(let m):
                state.isLoading = false
                state.messages = m
                return .none

            case .loadFailed(let msg):
                state.isLoading = false
                state.errorMessage = msg
                return .none

            case .sendTapped:
                let body = state.draft.trimmingCharacters(in: .whitespacesAndNewlines)
                guard !body.isEmpty, !state.isSending else { return .none }
                state.isSending = true
                let id = state.channelId
                state.draft = ""
                return .run { send in
                    do {
                        let payload = try JSONEncoder().encode(["source": body])
                        let json = String(data: payload, encoding: .utf8) ?? "{}"
                        let msg = try await core.sendChannelMessage(id, json, nil, nil)
                        await send(.messageSent(msg))
                    } catch {
                        await send(.sendFailed(error.localizedDescription))
                    }
                }

            case .messageSent(let msg):
                state.isSending = false
                state.messages.append(msg)
                return .none

            case .sendFailed(let msg):
                state.isSending = false
                state.errorMessage = msg
                return .none

            case .binding:
                return .none
            }
        }
    }
}
