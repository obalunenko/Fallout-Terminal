package player

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/coder/websocket"
)

const defaultConnectionQueueSize = 32

// PlayerConnection owns one socket reader and exactly one socket writer.
// Outbound producers never perform network I/O and never block canonical live
// state; a full queue closes only that slow connection.
type PlayerConnection struct {
	id     string
	socket *websocket.Conn
	send   chan []byte

	startOnce sync.Once
	closeOnce sync.Once
	mu        sync.Mutex
	cancel    context.CancelFunc
	done      chan struct{}
	workers   sync.WaitGroup
}

// NewPlayerConnection constructs a dormant connection. Start must be called
// once after the server registers it.
func NewPlayerConnection(id string, socket *websocket.Conn, queueSize int) *PlayerConnection {
	if queueSize <= 0 {
		queueSize = defaultConnectionQueueSize
	}
	return &PlayerConnection{
		id: id, socket: socket, send: make(chan []byte, queueSize), done: make(chan struct{}),
	}
}

// ID returns the server-local opaque connection identifier.
func (connection *PlayerConnection) ID() string {
	if connection == nil {
		return ""
	}
	return connection.id
}

// Start launches one reader and one writer. Syntactically invalid messages are
// ignored; oversized input closes the offending connection.
func (connection *PlayerConnection) Start(ctx context.Context, onMessage func(ClientMessage)) {
	if connection == nil {
		return
	}
	connection.startOnce.Do(func() {
		if ctx == nil {
			ctx = context.Background()
		}
		workerContext, cancel := context.WithCancel(ctx)
		connection.mu.Lock()
		connection.cancel = cancel
		connection.mu.Unlock()
		if connection.socket == nil {
			cancel()
			close(connection.done)
			return
		}
		connection.socket.SetReadLimit(MaxClientMessageBytes + 1)
		connection.workers.Add(2)
		go connection.readLoop(workerContext, onMessage)
		go connection.writeLoop(workerContext)
		go func() {
			connection.workers.Wait()
			connection.Close()
			close(connection.done)
		}()
	})
}

// Send enqueues a detached text payload. False means the connection is closed
// or was closed now because its bounded queue overflowed.
func (connection *PlayerConnection) Send(payload []byte) bool {
	if connection == nil {
		return false
	}
	copy := append([]byte(nil), payload...)
	select {
	case <-connection.done:
		return false
	default:
	}
	select {
	case connection.send <- copy:
		return true
	default:
		connection.Close()
		return false
	}
}

// Close cancels both loops and closes the socket immediately. It is safe from
// every goroutine and on repeated calls.
func (connection *PlayerConnection) Close() {
	if connection == nil {
		return
	}
	connection.closeOnce.Do(func() {
		connection.mu.Lock()
		cancel := connection.cancel
		connection.mu.Unlock()
		if cancel != nil {
			cancel()
		}
		if connection.socket != nil {
			_ = connection.socket.CloseNow()
		}
	})
}

// Done closes after both owned loops exit.
func (connection *PlayerConnection) Done() <-chan struct{} {
	if connection == nil {
		done := make(chan struct{})
		close(done)
		return done
	}
	return connection.done
}

func (connection *PlayerConnection) readLoop(ctx context.Context, onMessage func(ClientMessage)) {
	defer connection.workers.Done()
	defer connection.Close()
	for {
		messageType, reader, err := connection.socket.Reader(ctx)
		if err != nil {
			return
		}
		if messageType != websocket.MessageText {
			continue
		}
		message, err := DecodeClientMessage(reader)
		if err != nil {
			if errors.Is(err, ErrMessageTooLarge) {
				return
			}
			continue
		}
		if onMessage != nil {
			onMessage(message)
		}
	}
}

func (connection *PlayerConnection) writeLoop(ctx context.Context) {
	defer connection.workers.Done()
	defer connection.Close()
	for {
		select {
		case <-ctx.Done():
			return
		case payload := <-connection.send:
			if err := connection.socket.Write(ctx, websocket.MessageText, payload); err != nil {
				return
			}
		}
	}
}

// sameHostOrigin accepts non-browser clients without Origin and browser
// clients whose HTTP(S) origin host exactly matches the request host. This
// naturally permits both the local address and an authenticated ngrok host
// while rejecting arbitrary cross-origin pages.
func sameHostOrigin(request *http.Request) bool {
	if request == nil {
		return false
	}
	rawOrigin := strings.TrimSpace(request.Header.Get("Origin"))
	if rawOrigin == "" {
		return true
	}
	origin, err := url.Parse(rawOrigin)
	if err != nil || (origin.Scheme != "http" && origin.Scheme != "https") || origin.Host == "" {
		return false
	}
	return strings.EqualFold(origin.Host, request.Host)
}
