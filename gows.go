package gows

import (
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	gorillaws "github.com/gorilla/websocket"
)

// ErrHandshake is returned if a request can not be
// upgraded to a websocket connection
var ErrHandshake = errors.New("websocket handshake error")

// ErrConnectionClosed is returned if trying to perform an operation on a closed connection
var ErrConnectionClosed = errors.New("websocket connection closed")

// Websocket represents a websocket connection on which
// operations can be performed.
type Websocket interface {
	// WriteText writes data as type Text over the websocket connection
	WriteText(data []byte) error
	// WriteBinary writes data as type Binary over the websocket connection
	WriteBinary(data []byte) error
	// Read subscribes to messages that are sent from the client. If there are
	// more than a single reader, the message will be passed to ONE of the reader
	// at random.
	Read() <-chan Message
	// Close will close the websocket connection
	Close()
	// OnClose can be used to get a signal when the connection is closed. If
	// the connection was closed due to a failure, the error that caused the
	// closing will be returned
	OnClose() chan error
}

// MessageType represents a type of websocket message
type MessageType int

// MessageType constants
const (
	TextMessage MessageType = iota + 1
	BinaryMessage
)

// Message has the actual data and the corresponding type that the
// data was sent as
type Message struct {
	Type MessageType
	Data []byte
}

var upgrader = gorillaws.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	// allow any origin
	CheckOrigin: func(r *http.Request) bool { return true },
	// do not automatically write errors:
	Error: func(w http.ResponseWriter, r *http.Request, status int, reason error) {},
}

// Upgrade attempts to upgrade a HTTP request to a websocket connection. If the
// upgrade fails, an ErrHandshake error will be returned. Optional options can
// be passed to alter some details of the connection. See the available options
// to get their default values.
func Upgrade(w http.ResponseWriter, r *http.Request, opts ...Option) (Websocket, error) {
	ws := &websocket{
		options: defaultOptions,
	}

	for _, opt := range opts {
		opt(ws)
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrHandshake, err.Error())
	}

	ws.conn = conn
	ws.send = make(chan writeRequest)
	ws.reader = make(chan Message, 32)
	ws.onClose = make(chan error, 1)

	go ws.handleWriting()
	go ws.handleReading()

	return ws, nil
}

type websocket struct {
	conn *gorillaws.Conn

	send    chan writeRequest
	reader  chan Message
	onClose chan error

	closeOnce sync.Once
	closed    bool

	options options
}

type writeRequest struct {
	Type   MessageType
	Data   []byte
	Result chan<- error
}

func (ws *websocket) WriteText(data []byte) error {
	return ws.writeInternal(data, TextMessage)
}

func (ws *websocket) WriteBinary(data []byte) error {
	return ws.writeInternal(data, BinaryMessage)
}

func (ws *websocket) writeInternal(data []byte, messageType MessageType) error {
	if ws.closed {
		return ErrConnectionClosed
	}
	resultChan := make(chan error, 1)
	ws.send <- writeRequest{
		messageType,
		data,
		resultChan,
	}
	return <-resultChan
}

func (ws *websocket) Read() <-chan Message {
	return ws.reader
}

func (ws *websocket) Close() {
	ws.close(nil)
}

func (ws *websocket) OnClose() chan error {
	return ws.onClose
}

func (ws *websocket) close(reason error) {
	ws.closeOnce.Do(func() {
		ws.closed = true
		ws.conn.WriteMessage(gorillaws.CloseMessage, []byte{})
		ws.conn.Close()
		ws.onClose <- reason
		close(ws.reader)
		close(ws.send)
		close(ws.onClose)
	})
}

func (ws *websocket) broadcast(msg *Message) {
	ws.reader <- *msg
}

func (ws *websocket) handleReading() {
	ws.conn.SetReadLimit(ws.options.maxMessageSize)
	if !ws.options.disablePingPong {
		ws.conn.SetReadDeadline(time.Now().Add(ws.options.pongTimeout))
		ws.conn.SetPongHandler(func(string) error { ws.conn.SetReadDeadline(time.Now().Add(ws.options.pongTimeout)); return nil })
	} else {
		// Disable read timeout
		ws.conn.SetReadDeadline(time.Time{})
	}

	for {
		messageType, message, err := ws.conn.ReadMessage()
		if err != nil {
			ws.close(err)
			return
		}

		msg := &Message{
			Type: BinaryMessage,
			Data: message,
		}
		if messageType == gorillaws.TextMessage {
			msg.Type = TextMessage
		}
		go ws.broadcast(msg)
	}
}

func (ws *websocket) handleWriting() {
	ticker := time.NewTicker(ws.options.pingPeriod)

	defer ticker.Stop()

	for {
		select {
		case msg, ok := <-ws.send:
			if ws.options.writeTimeout > 0 {
				ws.conn.SetWriteDeadline(time.Now().Add(ws.options.writeTimeout))
			}
			if !ok {
				// the send channel was closed due to the connection being closed,
				// stop waiting for new writes
				return
			}

			messageType := gorillaws.BinaryMessage
			if msg.Type == TextMessage {
				messageType = gorillaws.TextMessage
			}
			err := ws.conn.WriteMessage(messageType, msg.Data)
			msg.Result <- err
		case <-ticker.C:
			if ws.options.disablePingPong {
				continue
			}
			if ws.options.writeTimeout > 0 {
				ws.conn.SetWriteDeadline(time.Now().Add(ws.options.writeTimeout))
			}
			if err := ws.conn.WriteMessage(gorillaws.PingMessage, nil); err != nil {
				ws.close(err)
				return
			}
		}
	}
}
