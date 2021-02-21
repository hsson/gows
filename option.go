package gows

import "time"

// Option is used to alter the default behavior of the
// websocket connection. Options can be optionally passed
// to the Upgrade function.
type Option func(ws *websocket)

type options struct {
	maxMessageSize  int64
	writeTimeout    time.Duration
	pongTimeout     time.Duration
	pingPeriod      time.Duration // must be less than pongTimeout
	disablePingPong bool
}

var defaultOptions = options{
	maxMessageSize:  2048,
	writeTimeout:    10 * time.Second,
	pongTimeout:     60 * time.Second,
	pingPeriod:      (60 * time.Second * 9) / 10,
	disablePingPong: false,
}

// MaxMessageSize defines the maximum size allowed for messages
// that are read from clients. If a client attempts to send a larger
// message, an error will occur.
//
// Default: 2048
func MaxMessageSize(size int64) Option {
	if size < 0 {
		panic("size must be non-negative")
	}
	return func(ws *websocket) {
		ws.options.maxMessageSize = size
	}
}

// WriteTimeout defines how long to wait for messages to be sent
// to the client before aborting. Setting it to zero means no timeout.
//
// Default: 10 seconds
func WriteTimeout(t time.Duration) Option {
	return func(ws *websocket) {
		ws.options.writeTimeout = t
	}
}

// PongTimeout defines how long to wait for a pong response before
// determining the connection to be dead. Ping messages will be sent
// in intervals from which pong responses are expected.
//
// Default: 60 seconds
func PongTimeout(t time.Duration) Option {
	if t == 0 {
		panic("timeout can not be zero")
	}
	return func(ws *websocket) {
		ws.options.pongTimeout = t
		ws.options.pingPeriod = (t * 9) / 10
	}
}

// DisablePingPong disables the pong/pong mecahnism and means that
// even if there is no communication the connection will be considered
// alive.
//
// Default: false
func DisablePingPong() Option {
	return func(ws *websocket) {
		ws.options.disablePingPong = true
	}
}
