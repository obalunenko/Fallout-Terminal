package player

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/coder/websocket"

	"github.com/obalunenko/Fallout-Terminal/internal/domain"
	"github.com/obalunenko/Fallout-Terminal/internal/live"
)

const defaultAddress = "0.0.0.0:3690"

// Config contains the player server's process-local dependencies. Assets must
// be rooted at client/; the server never opens player files from the host.
type Config struct {
	Address       string
	Assets        fs.FS
	Live          *live.Service
	QueueSize     int
	OnClientCount func(int)
	OnHackState   func(*domain.PublicHackState)
}

// Server owns the LAN HTTP listener, WebSocket registry, and live-state
// protocol dispatch. Canonical state remains exclusively owned by Live.
type Server struct {
	config Config

	mu         sync.Mutex
	listener   net.Listener
	httpServer *http.Server
	info       domain.ServerInfo
	clients    map[string]*PlayerConnection
	cancel     context.CancelFunc
	started    bool
	stopping   bool
	stopped    bool
	stopDone   chan struct{}
	stopErr    error

	clientSequence atomic.Uint64
	workers        sync.WaitGroup
}

// NewServer validates construction-only dependencies without acquiring the
// listener. Start is the sole resource acquisition boundary.
func NewServer(config Config) (*Server, error) {
	if config.Address == "" {
		config.Address = defaultAddress
	}
	if config.Assets == nil {
		return nil, errors.New("player server assets are not configured")
	}
	if config.Live == nil {
		return nil, errors.New("player live service is not configured")
	}
	if config.QueueSize <= 0 {
		config.QueueSize = defaultConnectionQueueSize
	}
	return &Server{config: config, clients: make(map[string]*PlayerConnection)}, nil
}

// Start acquires the listener before returning its usable local address.
func (server *Server) Start(_ context.Context) (domain.ServerInfo, error) {
	server.mu.Lock()
	defer server.mu.Unlock()
	if server.started {
		return server.info, nil
	}
	if server.stopped {
		return domain.ServerInfo{}, errors.New("player server cannot restart after shutdown")
	}

	listener, err := net.Listen(listenerNetwork(server.config.Address), server.config.Address)
	if err != nil {
		return domain.ServerInfo{}, fmt.Errorf("listen on %s: %w", server.config.Address, err)
	}
	serverContext, cancel := context.WithCancel(context.Background())
	server.listener = listener
	server.cancel = cancel
	server.info = listenerInfo(listener, server.config.Address)
	server.started = true
	server.stopDone = make(chan struct{})

	assets := NewHTTPHandler(server.config.Assets)
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(response http.ResponseWriter, request *http.Request) {
		if websocketUpgrade(request) {
			server.serveWebSocket(serverContext, response, request)
			return
		}
		assets.ServeHTTP(response, request)
	})
	server.httpServer = &http.Server{
		Handler: mux,
		BaseContext: func(net.Listener) context.Context {
			return serverContext
		},
	}
	server.workers.Add(1)
	go func(httpServer *http.Server, activeListener net.Listener) {
		defer server.workers.Done()
		_ = httpServer.Serve(activeListener)
	}(server.httpServer, listener)

	return server.info, nil
}

func listenerNetwork(address string) string {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return "tcp"
	}
	ip := net.ParseIP(host)
	if ip != nil && ip.To4() != nil {
		return "tcp4"
	}
	if ip != nil && strings.Contains(host, ":") {
		return "tcp6"
	}
	return "tcp"
}

// Info returns the detached address acquired by Start.
func (server *Server) Info() domain.ServerInfo {
	if server == nil {
		return domain.ServerInfo{}
	}
	server.mu.Lock()
	defer server.mu.Unlock()
	return server.info
}

// PublishLive sends the complete current state to every connected player.
func (server *Server) PublishLive() {
	if state := server.config.Live.Snapshot(); state != nil {
		server.broadcast(NewTerminalLiveEnvelope(state))
	}
}

// PublishUpdate sends a content update while retaining the current puzzle.
func (server *Server) PublishUpdate() {
	if state := server.config.Live.Snapshot(); state != nil {
		server.broadcast(NewTerminalUpdateEnvelope(state))
	}
}

// PublishClear tells every player to discard its current terminal.
func (server *Server) PublishClear() {
	server.broadcast(NewTerminalClearEnvelope())
}

// PublishHack sends the current public hacking state, if one exists.
func (server *Server) PublishHack() {
	if state := server.config.Live.Snapshot(); state != nil && state.Hack != nil {
		server.broadcast(NewHackStateEnvelope(state.Hack))
	}
}

// Stop closes clients and the listener. Concurrent and repeated calls observe
// the same shutdown result without holding the registry lock while waiting.
func (server *Server) Stop(ctx context.Context) error {
	if server == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	server.mu.Lock()
	if server.stopped {
		err := server.stopErr
		server.mu.Unlock()
		return err
	}
	if !server.started {
		server.stopped = true
		server.mu.Unlock()
		return nil
	}
	if server.stopping {
		done := server.stopDone
		server.mu.Unlock()
		select {
		case <-done:
			server.mu.Lock()
			err := server.stopErr
			server.mu.Unlock()
			return err
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	server.stopping = true
	cancel := server.cancel
	httpServer := server.httpServer
	clients := make([]*PlayerConnection, 0, len(server.clients))
	for _, connection := range server.clients {
		clients = append(clients, connection)
	}
	server.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	for _, connection := range clients {
		connection.Close()
	}
	var shutdownErr error
	if httpServer != nil {
		shutdownErr = httpServer.Shutdown(ctx)
	}
	waitDone := make(chan struct{})
	go func() {
		server.workers.Wait()
		close(waitDone)
	}()
	select {
	case <-waitDone:
	case <-ctx.Done():
		shutdownErr = errors.Join(shutdownErr, ctx.Err())
	}

	server.mu.Lock()
	server.started = false
	server.stopping = false
	server.stopped = true
	server.stopErr = shutdownErr
	close(server.stopDone)
	server.mu.Unlock()
	return shutdownErr
}

func (server *Server) serveWebSocket(ctx context.Context, response http.ResponseWriter, request *http.Request) {
	if !sameHostOrigin(request) {
		http.Error(response, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}
	socket, err := websocket.Accept(response, request, nil)
	if err != nil {
		return
	}

	id := strconv.FormatUint(server.clientSequence.Add(1), 10)
	connection := NewPlayerConnection(id, socket, server.config.QueueSize)
	server.mu.Lock()
	if server.stopping || server.stopped {
		server.mu.Unlock()
		connection.Close()
		return
	}
	server.clients[id] = connection
	count := len(server.clients)
	server.mu.Unlock()
	server.emitClientCount(count)

	if state := server.config.Live.Snapshot(); state != nil {
		payload, marshalErr := json.Marshal(NewTerminalLiveEnvelope(state))
		if marshalErr != nil || !connection.Send(payload) {
			connection.Close()
		}
	}
	connection.Start(ctx, server.handleClientMessage)
	server.workers.Add(1)
	go func() {
		defer server.workers.Done()
		<-connection.Done()
		server.removeClient(connection)
	}()
}

func (server *Server) handleClientMessage(message ClientMessage) {
	switch message.Type {
	case MessageNavAction:
		if state, ok := server.config.Live.ApplyNav(message.Action, message.NodeID); ok {
			server.broadcast(NewNavStateEnvelope(state))
		}
	case MessageHackGuess:
		if state, ok := server.config.Live.ApplyHackGuess(message.TargetID); ok {
			server.broadcast(NewHackStateEnvelope(state))
			server.emitHackState(state)
		}
	case MessageHackPattern:
		server.config.Live.ApplyHackPattern(message.PatternID, func(state *domain.PublicHackState) {
			server.broadcast(NewHackStateEnvelope(state))
			server.emitHackState(state)
		})
	}
}

func (server *Server) broadcast(envelope any) {
	payload, err := json.Marshal(envelope)
	if err != nil {
		return
	}
	server.mu.Lock()
	clients := make([]*PlayerConnection, 0, len(server.clients))
	for _, connection := range server.clients {
		clients = append(clients, connection)
	}
	server.mu.Unlock()
	for _, connection := range clients {
		connection.Send(payload)
	}
}

func (server *Server) removeClient(connection *PlayerConnection) {
	server.mu.Lock()
	if current, exists := server.clients[connection.ID()]; !exists || current != connection {
		server.mu.Unlock()
		return
	}
	delete(server.clients, connection.ID())
	count := len(server.clients)
	server.mu.Unlock()
	server.emitClientCount(count)
}

func (server *Server) emitClientCount(count int) {
	if server.config.OnClientCount != nil {
		server.config.OnClientCount(count)
	}
}

func (server *Server) emitHackState(state *domain.PublicHackState) {
	if server.config.OnHackState != nil {
		server.config.OnHackState(state)
	}
}

func websocketUpgrade(request *http.Request) bool {
	return request != nil && request.Method == http.MethodGet &&
		request.Header.Get("Sec-WebSocket-Key") != "" &&
		request.Header.Get("Upgrade") != ""
}

func listenerInfo(listener net.Listener, configuredAddress string) domain.ServerInfo {
	tcpAddress, _ := listener.Addr().(*net.TCPAddr)
	port := 0
	if tcpAddress != nil {
		port = tcpAddress.Port
	}
	host, _, err := net.SplitHostPort(configuredAddress)
	if err != nil || host == "" || host == "0.0.0.0" || host == "::" {
		host = localIPv4()
	}
	localHost := host
	if localHost != "127.0.0.1" && localHost != "::1" {
		localHost = "127.0.0.1"
	}
	return domain.ServerInfo{
		IP:       host,
		Port:     port,
		URL:      (&url.URL{Scheme: "http", Host: net.JoinHostPort(host, strconv.Itoa(port))}).String(),
		LocalURL: (&url.URL{Scheme: "http", Host: net.JoinHostPort(localHost, strconv.Itoa(port))}).String(),
	}
}

func localIPv4() string {
	addresses, err := net.InterfaceAddrs()
	if err == nil {
		for _, address := range addresses {
			ip, _, parseErr := net.ParseCIDR(address.String())
			if parseErr == nil && ip.To4() != nil && !ip.IsLoopback() {
				return ip.String()
			}
		}
	}
	return "127.0.0.1"
}
