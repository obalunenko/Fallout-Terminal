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

	"github.com/obalunenko/Fallout-Terminal/internal/control"
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
	Coordinator   PlayerCoordinator
	QueueSize     int
	OnClientCount func(int)
	OnHackState   func(*domain.PublicHackState)
}

// PlayerCoordinator is the process-local identity and assignment boundary
// used by the WebSocket server. Its methods enqueue detached coordination
// effects synchronously; they never write to sockets themselves.
type PlayerCoordinator interface {
	AttachConnection(domain.ConnectionID, domain.BrowserToken) (domain.BrowserToken, *domain.PlayerState)
	DetachConnection(domain.ConnectionID)
	SelectCharacter(domain.ConnectionID, string, domain.BroadcastID, domain.CharacterID)
}

// playerActionCoordinator is the US2 extension to the identity and assignment
// seam. Keeping it additive preserves the US1 transport boundary and the
// coordinator-free compatibility path while allowing every decoded shared
// action to retain its concrete sender identity.
type playerActionCoordinator interface {
	DispatchPlayerAction(domain.ConnectionID, domain.RuntimeCommand)
}

type playerLiveSnapshotCoordinator interface {
	CurrentLiveForSession(domain.LogicalSessionID) (*domain.PublicLiveState, uint64, bool)
}

type liveDelivery struct {
	revision   uint64
	terminalID string
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

	sessionByConnection   map[string]domain.LogicalSessionID
	connectionsBySession  map[domain.LogicalSessionID]map[string]*PlayerConnection
	playerStateBySession  map[domain.LogicalSessionID]*domain.PlayerState
	liveDeliveryBySession map[domain.LogicalSessionID]liveDelivery

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
	return &Server{
		config:                config,
		clients:               make(map[string]*PlayerConnection),
		sessionByConnection:   make(map[string]domain.LogicalSessionID),
		connectionsBySession:  make(map[domain.LogicalSessionID]map[string]*PlayerConnection),
		playerStateBySession:  make(map[domain.LogicalSessionID]*domain.PlayerState),
		liveDeliveryBySession: make(map[domain.LogicalSessionID]liveDelivery),
	}, nil
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
		envelope := NewTerminalLiveEnvelope(0, state)
		if server.config.Coordinator != nil {
			server.sendToAssigned(envelope, state.TerminalID)
			return
		}
		server.broadcast(envelope)
	}
}

// PublishUpdate sends a content update while retaining the current puzzle.
func (server *Server) PublishUpdate() {
	if state := server.config.Live.Snapshot(); state != nil {
		envelope := NewTerminalUpdateEnvelope(0, state)
		if server.config.Coordinator != nil {
			server.sendToAssigned(envelope, state.TerminalID)
			return
		}
		server.broadcast(envelope)
	}
}

// PublishClear tells every player to discard its current terminal.
func (server *Server) PublishClear() {
	envelope := NewTerminalClearEnvelope(0)
	if server.config.Coordinator != nil {
		server.sendToAssigned(envelope, "")
		return
	}
	server.broadcast(envelope)
}

// PublishHack sends the current public hacking state, if one exists.
func (server *Server) PublishHack() {
	if state := server.config.Live.Snapshot(); state != nil && state.Hack != nil {
		envelope := NewHackStateEnvelope(0, state.TerminalID, state.Hack)
		if server.config.Coordinator != nil {
			server.sendToAssigned(envelope, state.TerminalID)
			return
		}
		server.broadcast(envelope)
	}
}

// PublishCoordinationEffect routes one already-ordered coordinator effect to
// its personalized player recipients. It performs only detached encoding and
// non-blocking queue sends, so it is safe for the coordinator to call before
// releasing its transaction mutex.
func (server *Server) PublishCoordinationEffect(effect control.Effect) {
	if server == nil || server.config.Coordinator == nil {
		return
	}
	server.publishPersonalizedCoordination(effect)
	if effect.ClearLiveTerminal {
		server.publishClearEffect(effect)
	}
	if effect.Live != nil {
		server.publishLiveEffect(effect.SessionID, effect.Live, effect.Revision)
	}
	if effect.Hack != nil && effect.TerminalID != "" {
		server.sendToAssigned(NewHackStateEnvelope(effect.Revision, effect.TerminalID, effect.Hack), effect.TerminalID)
	}
	if effect.Result != nil && effect.ConnectionID != "" {
		server.sendToConnection(string(effect.ConnectionID), NewActionResultEnvelope(*effect.Result))
	}
}

// publishPersonalizedCoordination also carries player-config roster installs.
// The coordinator emits one complete state per recognized session, so a newly
// loaded authored roster reaches every open tab without importing any former
// process's token, claim, controller, broadcast, terminal, or puzzle state.
func (server *Server) publishPersonalizedCoordination(effect control.Effect) {
	if effect.Player == nil || effect.SessionID == "" || effect.Player.SessionID != effect.SessionID {
		return
	}
	server.publishPlayerState(effect.SessionID, effect.Player, effect.Revision)
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

	// The legacy path remains available while the coordinator is not composed.
	// Once coordination is configured, absolutely no roster or terminal state
	// crosses the socket before SESSION_HELLO succeeds.
	if server.config.Coordinator == nil {
		if state := server.config.Live.Snapshot(); state != nil {
			payload, marshalErr := json.Marshal(NewTerminalLiveEnvelope(0, state))
			if marshalErr != nil || !connection.Send(payload) {
				connection.Close()
			}
		}
	}
	connection.StartWithSender(ctx, server.handleClientMessage)
	server.workers.Add(1)
	go func() {
		defer server.workers.Done()
		<-connection.Done()
		server.removeClient(connection)
	}()
}

func (server *Server) handleClientMessage(connection *PlayerConnection, message ClientMessage) {
	if server.config.Coordinator != nil {
		if message.Type == MessageSessionHello {
			server.handleSessionHello(connection, message)
			return
		}
		_, handshaken := server.boundSession(connection)
		if !handshaken {
			return
		}
		if message.Type == MessageCharacterSelect {
			server.config.Coordinator.SelectCharacter(
				domain.ConnectionID(connection.ID()),
				message.RequestID,
				domain.BroadcastID(message.BroadcastID),
				domain.CharacterID(message.CharacterID),
			)
			return
		}

		actions, ok := server.config.Coordinator.(playerActionCoordinator)
		if !ok {
			return
		}
		command, ok := runtimeCommand(message)
		if !ok {
			return
		}
		actions.DispatchPlayerAction(domain.ConnectionID(connection.ID()), command)
		return
	}

	server.handleLegacyClientMessage(message)
}

func runtimeCommand(message ClientMessage) (domain.RuntimeCommand, bool) {
	command := domain.RuntimeCommand{
		RequestID: domain.RequestID(message.RequestID), BroadcastID: domain.BroadcastID(message.BroadcastID),
		TerminalID: message.TerminalID,
	}
	switch message.Type {
	case MessageNavAction:
		command.Kind = domain.RuntimeCommandNavAction
		command.Action = message.Action
		command.NodeID = message.NodeID
	case MessageHackGuess:
		command.Kind = domain.RuntimeCommandHackGuess
		command.TargetID = message.TargetID
	case MessageHackPattern:
		command.Kind = domain.RuntimeCommandHackPattern
		command.PatternID = message.PatternID
	default:
		return domain.RuntimeCommand{}, false
	}
	return command, true
}

func (server *Server) handleLegacyClientMessage(message ClientMessage) {
	switch message.Type {
	case MessageNavAction:
		if state, ok := server.config.Live.ApplyNav(message.Action, message.NodeID); ok {
			terminalID := ""
			if liveState := server.config.Live.Snapshot(); liveState != nil {
				terminalID = liveState.TerminalID
			}
			server.broadcast(NewNavStateEnvelope(0, terminalID, state))
		}
	case MessageHackGuess:
		if state, ok := server.config.Live.ApplyHackGuess(message.TargetID); ok {
			terminalID := ""
			if liveState := server.config.Live.Snapshot(); liveState != nil {
				terminalID = liveState.TerminalID
			}
			server.broadcast(NewHackStateEnvelope(0, terminalID, state))
			server.emitHackState(state)
		}
	case MessageHackPattern:
		terminalID := ""
		if liveState := server.config.Live.Snapshot(); liveState != nil {
			terminalID = liveState.TerminalID
		}
		server.config.Live.ApplyHackPattern(message.PatternID, func(state *domain.PublicHackState) {
			server.broadcast(NewHackStateEnvelope(0, terminalID, state))
			server.emitHackState(state)
		})
	}
}

func (server *Server) handleSessionHello(connection *PlayerConnection, message ClientMessage) {
	if connection == nil || server.config.Coordinator == nil {
		return
	}
	if _, alreadyBound := server.boundSession(connection); alreadyBound {
		return
	}

	token, state := server.config.Coordinator.AttachConnection(
		domain.ConnectionID(connection.ID()),
		domain.BrowserToken(message.BrowserToken),
	)
	if token == "" || state == nil || state.SessionID == "" {
		connection.Close()
		return
	}
	if !server.bindConnection(connection, state) {
		server.config.Coordinator.DetachConnection(domain.ConnectionID(connection.ID()))
		return
	}
	if !server.sendEnvelope(connection, NewSessionWelcomeEnvelope(string(token), state)) {
		connection.Close()
		return
	}
	server.deliverCurrentLiveToConnection(connection, state, state.Revision)
}

func (server *Server) bindConnection(connection *PlayerConnection, state *domain.PlayerState) bool {
	if connection == nil || state == nil || state.SessionID == "" {
		return false
	}
	server.mu.Lock()
	defer server.mu.Unlock()
	if current, exists := server.clients[connection.ID()]; !exists || current != connection || server.stopping || server.stopped {
		return false
	}
	if _, exists := server.sessionByConnection[connection.ID()]; exists {
		return false
	}
	server.sessionByConnection[connection.ID()] = state.SessionID
	connections := server.connectionsBySession[state.SessionID]
	if connections == nil {
		connections = make(map[string]*PlayerConnection)
		server.connectionsBySession[state.SessionID] = connections
	}
	connections[connection.ID()] = connection
	server.playerStateBySession[state.SessionID] = domain.ClonePlayerState(state)
	return true
}

func (server *Server) boundSession(connection *PlayerConnection) (domain.LogicalSessionID, bool) {
	if connection == nil {
		return "", false
	}
	server.mu.Lock()
	defer server.mu.Unlock()
	sessionID, ok := server.sessionByConnection[connection.ID()]
	return sessionID, ok
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

func (server *Server) publishPlayerState(sessionID domain.LogicalSessionID, state *domain.PlayerState, revision uint64) {
	if state == nil || sessionID == "" || state.SessionID != sessionID {
		return
	}
	detached := domain.ClonePlayerState(state)
	if revision != 0 {
		detached.Revision = revision
	}
	server.mu.Lock()
	previous := server.playerStateBySession[sessionID]
	if previous != nil && detached.Revision < previous.Revision {
		server.mu.Unlock()
		return
	}
	needsLive := playerStateNeedsTerminalSnapshot(previous, detached)
	server.playerStateBySession[sessionID] = domain.ClonePlayerState(detached)
	connections := cloneConnections(server.connectionsBySession[sessionID])
	server.mu.Unlock()

	server.sendToConnections(connections, NewPlayerStateEnvelope(detached))
	if needsLive {
		server.deliverCurrentLive(sessionID, detached, detached.Revision)
	}
}

// playerStateNeedsTerminalSnapshot distinguishes a terminal-identity change
// from a role-only coordination revision. Reassignment effects therefore fan
// PLAYER_STATE to every tab of the former and new controller sessions without
// replaying unchanged terminal content; a later canonical live effect still
// enters each connection's writer queue after that role revision.
func playerStateNeedsTerminalSnapshot(previous, current *domain.PlayerState) bool {
	if current == nil || current.Character == nil || current.ActiveTerminalID == "" {
		return false
	}
	return previous == nil || previous.Character == nil || previous.ActiveTerminalID != current.ActiveTerminalID
}

func (server *Server) publishLiveEffect(sessionID domain.LogicalSessionID, state *domain.PublicLiveState, revision uint64) {
	if state == nil {
		return
	}
	if sessionID != "" {
		server.deliverLiveToSession(sessionID, state, revision)
		return
	}
	server.mu.Lock()
	sessionIDs := make([]domain.LogicalSessionID, 0, len(server.playerStateBySession))
	for candidate, playerState := range server.playerStateBySession {
		if playerState != nil && playerState.Character != nil && playerState.ActiveTerminalID == state.TerminalID {
			sessionIDs = append(sessionIDs, candidate)
		}
	}
	server.mu.Unlock()
	for _, candidate := range sessionIDs {
		server.deliverLiveToSession(candidate, state, revision)
	}
}

func (server *Server) publishClearEffect(effect control.Effect) {
	if effect.SessionID != "" {
		server.mu.Lock()
		state := server.playerStateBySession[effect.SessionID]
		if state == nil || state.Character == nil || effect.Revision < state.Revision {
			server.mu.Unlock()
			return
		}
		connections := cloneConnections(server.connectionsBySession[effect.SessionID])
		delete(server.liveDeliveryBySession, effect.SessionID)
		server.mu.Unlock()
		server.sendToConnections(connections, NewTerminalClearEnvelope(effect.Revision))
		return
	}
	server.mu.Lock()
	connections := make([]*PlayerConnection, 0)
	for sessionID, state := range server.playerStateBySession {
		_, hadLiveDelivery := server.liveDeliveryBySession[sessionID]
		if state == nil || effect.Revision < state.Revision || (!hadLiveDelivery && state.Character == nil) {
			continue
		}
		connections = append(connections, cloneConnections(server.connectionsBySession[sessionID])...)
		delete(server.liveDeliveryBySession, sessionID)
	}
	server.mu.Unlock()
	server.sendToConnections(connections, NewTerminalClearEnvelope(effect.Revision))
}

func (server *Server) deliverCurrentLive(sessionID domain.LogicalSessionID, state *domain.PlayerState, revision uint64) {
	// Coordinator transitions publish their detached live effect explicitly
	// after the personalized state effect. Querying the coordinator from this
	// synchronous effect callback would re-enter its transaction lock.
	if _, coordinated := server.config.Coordinator.(playerLiveSnapshotCoordinator); coordinated {
		return
	}
	if state == nil || state.Character == nil || state.ActiveTerminalID == "" || server.config.Live == nil {
		return
	}
	liveState := server.config.Live.Snapshot()
	if liveState == nil || liveState.TerminalID != state.ActiveTerminalID {
		return
	}
	server.deliverLiveToSession(sessionID, liveState, revision)
}

func (server *Server) deliverCurrentLiveToConnection(connection *PlayerConnection, state *domain.PlayerState, revision uint64) {
	if connection == nil || state == nil {
		return
	}
	liveState, snapshotRevision, ok := server.currentLiveSnapshot(state.SessionID, state, revision)
	if !ok {
		return
	}
	if server.sendEnvelope(connection, NewTerminalLiveEnvelope(snapshotRevision, liveState)) {
		server.mu.Lock()
		server.liveDeliveryBySession[state.SessionID] = liveDelivery{revision: snapshotRevision, terminalID: liveState.TerminalID}
		server.mu.Unlock()
	}
}

func (server *Server) currentLiveSnapshot(sessionID domain.LogicalSessionID, state *domain.PlayerState, revision uint64) (*domain.PublicLiveState, uint64, bool) {
	if state == nil || state.Character == nil || state.ActiveTerminalID == "" {
		return nil, revision, false
	}
	if coordinator, ok := server.config.Coordinator.(playerLiveSnapshotCoordinator); ok {
		liveState, snapshotRevision, available := coordinator.CurrentLiveForSession(sessionID)
		if !available || liveState == nil || liveState.TerminalID != state.ActiveTerminalID {
			return nil, snapshotRevision, false
		}
		return liveState, snapshotRevision, true
	}
	if server.config.Live == nil {
		return nil, revision, false
	}
	liveState := server.config.Live.Snapshot()
	if liveState == nil || liveState.TerminalID != state.ActiveTerminalID {
		return nil, revision, false
	}
	return liveState, revision, true
}

func (server *Server) deliverLiveToSession(sessionID domain.LogicalSessionID, state *domain.PublicLiveState, revision uint64) {
	if state == nil || sessionID == "" {
		return
	}
	server.mu.Lock()
	playerState := server.playerStateBySession[sessionID]
	if playerState == nil || playerState.Character == nil || playerState.ActiveTerminalID != state.TerminalID || revision < playerState.Revision {
		server.mu.Unlock()
		return
	}
	delivery := liveDelivery{revision: revision, terminalID: state.TerminalID}
	if server.liveDeliveryBySession[sessionID] == delivery {
		server.mu.Unlock()
		return
	}
	server.liveDeliveryBySession[sessionID] = delivery
	connections := cloneConnections(server.connectionsBySession[sessionID])
	server.mu.Unlock()
	server.sendToConnections(connections, NewTerminalLiveEnvelope(revision, state))
}

// sendToAssigned retains the legacy bridge envelope families while applying
// the coordinated authorization projection at the transport boundary. An
// empty terminal ID targets every assigned session (clear); otherwise only
// sessions currently assigned to that active terminal receive the envelope.
func (server *Server) sendToAssigned(envelope any, terminalID string) {
	server.mu.Lock()
	connections := make([]*PlayerConnection, 0)
	for sessionID, state := range server.playerStateBySession {
		if state == nil || state.Character == nil {
			continue
		}
		if terminalID != "" && state.ActiveTerminalID != terminalID {
			continue
		}
		connections = append(connections, cloneConnections(server.connectionsBySession[sessionID])...)
	}
	server.mu.Unlock()
	server.sendToConnections(connections, envelope)
}

func (server *Server) sendToConnection(connectionID string, envelope any) {
	server.mu.Lock()
	connection := server.clients[connectionID]
	server.mu.Unlock()
	server.sendEnvelope(connection, envelope)
}

func (server *Server) sendToConnections(connections []*PlayerConnection, envelope any) {
	payload, err := json.Marshal(envelope)
	if err != nil {
		return
	}
	for _, connection := range connections {
		connection.Send(payload)
	}
}

func (server *Server) sendEnvelope(connection *PlayerConnection, envelope any) bool {
	if connection == nil {
		return false
	}
	payload, err := json.Marshal(envelope)
	return err == nil && connection.Send(payload)
}

func cloneConnections(source map[string]*PlayerConnection) []*PlayerConnection {
	connections := make([]*PlayerConnection, 0, len(source))
	for _, connection := range source {
		connections = append(connections, connection)
	}
	return connections
}

func (server *Server) removeClient(connection *PlayerConnection) {
	server.mu.Lock()
	if current, exists := server.clients[connection.ID()]; !exists || current != connection {
		server.mu.Unlock()
		return
	}
	delete(server.clients, connection.ID())
	sessionID, bound := server.sessionByConnection[connection.ID()]
	delete(server.sessionByConnection, connection.ID())
	if bound {
		if connections := server.connectionsBySession[sessionID]; connections != nil {
			delete(connections, connection.ID())
			if len(connections) == 0 {
				delete(server.connectionsBySession, sessionID)
			}
		}
	}
	count := len(server.clients)
	server.mu.Unlock()
	if bound && server.config.Coordinator != nil {
		server.config.Coordinator.DetachConnection(domain.ConnectionID(connection.ID()))
	}
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
