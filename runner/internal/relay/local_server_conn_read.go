package relay

import (
	"time"

	"github.com/gorilla/websocket"
)

func (s *LocalServer) readLoop(podKey string, lane *localPodLane, client *localConn) {
	defer func() {
		client.close()
		s.logger.Info("Local relay browser disconnected", "pod_key", podKey)
	}()

	for {
		messageType, data, err := client.socket.ReadMessage()
		if err != nil {
			return
		}
		_ = client.socket.SetReadDeadline(time.Now().Add(localReadTimeout))
		if messageType != websocket.BinaryMessage || len(data) < 1 {
			continue
		}
		msgType := data[0]
		payload := data[1:]
		lane.mu.RLock()
		requestHandler := lane.reqHandlers[msgType]
		handler := lane.handlers[msgType]
		lane.mu.RUnlock()
		if requestHandler != nil {
			requestHandler(payload, func(replyType byte, replyPayload []byte) {
				client.enqueue(EncodeMessage(replyType, replyPayload))
			})
		} else if handler != nil {
			handler(payload)
		}
	}
}
