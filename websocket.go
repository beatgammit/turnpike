package turnpike

import (
	"fmt"
	"sync"

	"github.com/gorilla/websocket"
)

type websocketPeer struct {
	conn        *websocket.Conn
	serializer  Serializer
	messages    chan Message
	payloadType int
	mutex       sync.Mutex
}

// NewWebsocketPeer connects to the websocket server at the specified url.
func NewWebsocketPeer(serialization Serialization, url, origin string) (Peer, error) {
	switch serialization {
	case JSON:
		return newWebsocketPeer(url, jsonWebsocketProtocol, origin,
			new(JSONSerializer), websocket.TextMessage,
		)
	case MSGPACK:
		return newWebsocketPeer(url, msgpackWebsocketProtocol, origin,
			new(MessagePackSerializer), websocket.BinaryMessage,
		)
	default:
		return nil, fmt.Errorf("Unsupported serialization: %s", serialization)
	}
}

func newWebsocketPeer(url, protocol, origin string, serializer Serializer, payloadType int) (Peer, error) {
	dialer := websocket.Dialer{
		Subprotocols: []string{protocol},
	}
	conn, _, err := dialer.Dial(url, nil)
	if err != nil {
		return nil, err
	}
	ep := &websocketPeer{
		conn:        conn,
		messages:    make(chan Message, 10),
		serializer:  serializer,
		payloadType: payloadType,
	}
	go func() {
		for {
			// TODO: use conn.NextMessage() and stream
			// TODO: do something different based on binary/text frames
			if _, b, err := conn.ReadMessage(); err != nil {
				conn.Close()
				break
			} else {
				msg, err := serializer.Deserialize(b)
				if err != nil {
					// TODO: handle error
				} else {
					ep.messages <- msg
				}
			}
		}
	}()
	return ep, nil
}

func (ep *websocketPeer) Send(msg Message) error {
	b, err := ep.serializer.Serialize(msg)
	if err != nil {
		return err
	}
	ep.mutex.Lock()
	defer ep.mutex.Unlock()
	return ep.conn.WriteMessage(ep.payloadType, b)
}
func (ep *websocketPeer) Receive() <-chan Message {
	return ep.messages
}
func (ep *websocketPeer) Close() error {
	return ep.conn.Close()
}
