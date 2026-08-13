package main

import (
	"context"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/obalunenko/Fallout-Terminal/internal/control"
	"github.com/obalunenko/Fallout-Terminal/internal/domain"
	"github.com/obalunenko/Fallout-Terminal/internal/live"
	"github.com/obalunenko/Fallout-Terminal/internal/player"
)

const fixtureAddress = "127.0.0.1:34119"

type ids struct{ next atomic.Uint64 }

func (source *ids) Next() string {
	return fmt.Sprintf("browser-fixture-%d", source.next.Add(1))
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	playerAssets, err := fs.Sub(os.DirFS("../../client"), "dist")
	if err != nil {
		return fmt.Errorf("open built player assets: %w", err)
	}
	liveService := live.New(nil, nil)
	var connectPlayer *player.ConnectService
	service := control.New(control.Config{
		IDs:         &ids{},
		Runtime:     liveService,
		Terminals:   liveService,
		TrustedHack: liveService,
		Enqueue: func(effect control.Effect) {
			if connectPlayer != nil {
				connectPlayer.PublishEffect(effect)
			}
		},
	})
	connectPlayer, err = player.NewConnectService(player.ConnectServiceConfig{Coordinator: service, Assets: playerAssets})
	if err != nil {
		return fmt.Errorf("construct fixture Connect service: %w", err)
	}
	rpcPath, rpcHandler := player.NewConnectHandler(connectPlayer)
	applicationHandler := player.NewApplicationHandler(playerAssets, rpcPath, rpcHandler)

	for _, name := range []string{"Mara", "Boone"} {
		if _, err := service.AddCharacter(name); err != nil {
			return err
		}
	}
	if _, err := service.StartBroadcast(); err != nil {
		return err
	}
	if _, err := service.RequestTerminalActivation(fixtureTerminal()); err != nil {
		return err
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /__fixture/reset", func(response http.ResponseWriter, _ *http.Request) {
		if _, err := service.EndBroadcast(); err != nil {
			http.Error(response, err.Error(), http.StatusConflict)
			return
		}
		if _, err := service.StartBroadcast(); err != nil {
			http.Error(response, err.Error(), http.StatusInternalServerError)
			return
		}
		if _, err := service.RequestTerminalActivation(fixtureTerminal()); err != nil {
			http.Error(response, err.Error(), http.StatusInternalServerError)
			return
		}
		response.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("/__fixture/protected/", func(response http.ResponseWriter, request *http.Request) {
		username, password, ok := request.BasicAuth()
		if !ok || username != "players" || password != "password-long-enough" {
			response.Header().Set("WWW-Authenticate", `Basic realm="Fallout Terminal Players"`)
			http.Error(response, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		forwarded := request.Clone(request.Context())
		forwarded.URL.Path = "/" + strings.TrimPrefix(request.URL.Path, "/__fixture/protected/")
		applicationHandler.ServeHTTP(response, forwarded)
	})
	mux.Handle("/", applicationHandler)

	listener, err := net.Listen("tcp4", fixtureAddress)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", fixtureAddress, err)
	}
	httpServer := &http.Server{Handler: mux}
	serveErrors := make(chan error, 1)
	go func() {
		serveErrors <- httpServer.Serve(listener)
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	select {
	case <-ctx.Done():
	case err := <-serveErrors:
		if err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	}

	shutdownContext, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return httpServer.Shutdown(shutdownContext)
}

func fixtureTerminal() domain.TerminalTarget {
	return domain.TerminalTarget{
		TerminalID: "terminal-1", TerminalName: "Overseer", HackLevel: 0, IntroText: "WELCOME",
		Tree: domain.ContentNode{
			ID: "root", Type: domain.NodeFolder, Name: "ROOT",
			Children: []domain.ContentNode{
				{ID: "docs", Type: domain.NodeFolder, Name: "DOCS", Children: []domain.ContentNode{
					{ID: "report", Type: domain.NodeEntry, Name: "REPORT", Description: "SYSTEM NOMINAL"},
				}},
				{ID: "status", Type: domain.NodeEntry, Name: "STATUS", Description: "ALL SYSTEMS OPERATIONAL"},
			},
		},
	}
}
