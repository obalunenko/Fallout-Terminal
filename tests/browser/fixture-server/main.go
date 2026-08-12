package main

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/obalunenko/Fallout-Terminal/internal/control"
	"github.com/obalunenko/Fallout-Terminal/internal/domain"
	"github.com/obalunenko/Fallout-Terminal/internal/live"
	"github.com/obalunenko/Fallout-Terminal/internal/player"
)

type ids struct{ next atomic.Uint64 }

func (source *ids) Next() string {
	return fmt.Sprintf("browser-fixture-%d", source.next.Add(1))
}

type coordinatorAdapter struct {
	mu                  sync.RWMutex
	service             *control.Service
	sessionByConnection map[domain.ConnectionID]domain.LogicalSessionID
}

func (adapter *coordinatorAdapter) AttachConnection(connectionID domain.ConnectionID, token domain.BrowserToken) (domain.BrowserToken, *domain.PlayerState) {
	returnedToken, state := adapter.service.AttachConnection(connectionID, token)
	if state != nil {
		adapter.mu.Lock()
		adapter.sessionByConnection[connectionID] = state.SessionID
		adapter.mu.Unlock()
	}
	return returnedToken, state
}

func (adapter *coordinatorAdapter) DetachConnection(connectionID domain.ConnectionID) {
	adapter.mu.Lock()
	delete(adapter.sessionByConnection, connectionID)
	adapter.mu.Unlock()
	adapter.service.DetachConnection(connectionID)
}

func (adapter *coordinatorAdapter) SelectCharacter(connectionID domain.ConnectionID, requestID string, broadcastID domain.BroadcastID, characterID domain.CharacterID) {
	adapter.mu.RLock()
	sessionID := adapter.sessionByConnection[connectionID]
	adapter.mu.RUnlock()
	adapter.service.SelectCharacter(control.CharacterSelection{
		ConnectionID: connectionID,
		SessionID:    sessionID,
		RequestID:    domain.RequestID(requestID),
		BroadcastID:  broadcastID,
		CharacterID:  characterID,
	})
}

func (adapter *coordinatorAdapter) DispatchPlayerAction(connectionID domain.ConnectionID, command domain.RuntimeCommand) {
	adapter.service.DispatchPlayerAction(connectionID, command)
}

func (adapter *coordinatorAdapter) CurrentLiveForSession(sessionID domain.LogicalSessionID) (*domain.PublicLiveState, uint64, bool) {
	return adapter.service.CurrentLiveForSession(sessionID)
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	playerAssets := fs.FS(os.DirFS("../../client"))
	liveService := live.New(nil, nil)
	var playerServer *player.Server
	service := control.New(control.Config{
		IDs:         &ids{},
		Runtime:     liveService,
		Terminals:   liveService,
		TrustedHack: liveService,
		Enqueue: func(effect control.Effect) {
			if playerServer != nil {
				playerServer.PublishCoordinationEffect(effect)
			}
		},
	})
	adapter := &coordinatorAdapter{
		service:             service,
		sessionByConnection: make(map[domain.ConnectionID]domain.LogicalSessionID),
	}

	var err error
	playerServer, err = player.NewServer(player.Config{
		Address: "127.0.0.1:34119", Assets: playerAssets, Live: liveService, Coordinator: adapter,
	})
	if err != nil {
		return err
	}
	for _, name := range []string{"Mara", "Boone"} {
		if _, err := service.AddCharacter(name); err != nil {
			return err
		}
	}
	if _, err := service.StartBroadcast(); err != nil {
		return err
	}
	if _, err := service.RequestTerminalActivation(domain.TerminalTarget{
		TerminalID: "terminal-1", TerminalName: "Overseer", HackLevel: 1, IntroText: "WELCOME",
		Tree: domain.ContentNode{
			ID: "root", Type: domain.NodeFolder, Name: "ROOT",
			Children: []domain.ContentNode{{ID: "status", Type: domain.NodeEntry, Name: "STATUS", Description: "SYSTEM NOMINAL"}},
		},
	}); err != nil {
		return err
	}
	if _, err := playerServer.Start(context.Background()); err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	shutdownContext, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return playerServer.Stop(shutdownContext)
}
