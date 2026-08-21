package player

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/obalunenko/Fallout-Terminal/internal/control"
	"github.com/obalunenko/Fallout-Terminal/internal/domain"
	"github.com/obalunenko/logger"
)

const defaultAddress = "0.0.0.0:3690"

// Config contains the generated Connect player server's process-local
// dependencies. Assets are the complete built player application.
type Config struct {
	Address string
	Assets  fs.FS
	Connect *ConnectService
	Logger  logger.Logger
}

// Server owns the LAN HTTP listener and generated Connect stream lifecycle.
type Server struct {
	config Config
	log    logger.Logger

	mu         sync.Mutex
	listener   net.Listener
	httpServer *http.Server
	info       domain.ServerInfo
	cancel     context.CancelFunc
	started    bool
	stopping   bool
	stopped    bool
	stopDone   chan struct{}
	stopErr    error
	workers    sync.WaitGroup
}

// NewServer validates construction-only dependencies without acquiring a
// listener. A generated handler is mandatory: there is no legacy fallback.
func NewServer(config Config) (*Server, error) {
	if config.Address == "" {
		config.Address = defaultAddress
	}
	if config.Assets == nil {
		return nil, errors.New("player server assets are not configured")
	}
	if config.Connect == nil {
		return nil, errors.New("generated Connect player service is not configured")
	}
	serverLogger := config.Logger
	if serverLogger == nil {
		serverLogger = logger.FromContext(nil)
	}
	return &Server{config: config, log: serverLogger}, nil
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

	rpcPath, rpcHandler := NewConnectHandler(server.config.Connect)
	application := NewApplicationHandler(server.config.Assets, rpcPath, rpcHandler)
	server.httpServer = &http.Server{
		Handler: application,
		BaseContext: func(net.Listener) context.Context {
			return serverContext
		},
	}
	server.workers.Add(1)
	go func(httpServer *http.Server, activeListener net.Listener) {
		defer server.workers.Done()
		server.recordServeExit(httpServer.Serve(activeListener))
	}(server.httpServer, listener)
	return server.info, nil
}

func (server *Server) recordServeExit(err error) {
	if server == nil || server.log == nil || err == nil || errors.Is(err, http.ErrServerClosed) {
		return
	}
	server.log.WithError(err).WithField("operation", "player.serve").Error("player server stopped unexpectedly")
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

// PublishCoordinationEffect forwards only the coordinator's complete typed
// compound update. Component and request-result effects have no second wire
// representation.
func (server *Server) PublishCoordinationEffect(effect control.Effect) {
	if server != nil && server.config.Connect != nil {
		server.config.Connect.PublishEffect(effect)
	}
}

// Stop closes generated streams and the listener in bounded, idempotent order.
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
	server.mu.Unlock()

	if cancel != nil {
		cancel()
	}
	server.config.Connect.CloseSubscriptions()
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

func listenerInfo(listener net.Listener, configuredAddress string) domain.ServerInfo {
	tcpAddress, _ := listener.Addr().(*net.TCPAddr)
	port := 0
	if tcpAddress != nil {
		port = tcpAddress.Port
	}
	host, _, err := net.SplitHostPort(configuredAddress)
	if err != nil || host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	localURL := fmt.Sprintf("http://%s", net.JoinHostPort(host, fmt.Sprint(port)))
	return domain.ServerInfo{IP: host, URL: localURL, LocalURL: localURL, Port: port}
}
