package main

import (
	"context"
	"errors"
	"io/fs"
	"time"

	"github.com/obalunenko/Fallout-Terminal/internal/domain"
	"github.com/wailsapp/wails/v3/pkg/application"
)

const wailsShutdownTimeout = 5 * time.Second

func init() {
	application.RegisterEvent[domain.ServerInfo](serverInfoEvent)
	application.RegisterEvent[int](clientCountEvent)
	application.RegisterEvent[*domain.PublicHackState](hackStateEvent)
	application.RegisterEvent[*domain.MasterCoordinationState](coordinationStateEvent)
}

func newWailsApplication(frontendAssets fs.FS) *application.App {
	return application.New(wailsApplicationOptions(frontendAssets))
}

func wailsApplicationOptions(frontendAssets fs.FS) application.Options {
	return application.Options{
		Name:        "Fallout Terminal",
		Description: "Fallout Terminal — Master Control",
		Assets: application.AssetOptions{
			Handler: application.AssetFileServerFS(frontendAssets),
		},
		Mac: application.MacOptions{
			ApplicationShouldTerminateAfterLastWindowClosed: true,
		},
	}
}

func newMasterWindow(host *application.App) *application.WebviewWindow {
	return host.Window.NewWithOptions(masterWindowOptions())
}

func masterWindowOptions() application.WebviewWindowOptions {
	return application.WebviewWindowOptions{
		Title:            "Fallout Terminal — Master Control",
		Width:            1200,
		Height:           780,
		MinWidth:         900,
		MinHeight:        600,
		BackgroundColour: application.NewRGB(11, 13, 10),
		URL:              "/",
	}
}

// wailsServiceRegistrar is the narrow host seam needed after core composition.
// The concrete Wails application stays in root composition and the core App is
// never registered as a frontend service.
type wailsServiceRegistrar interface {
	RegisterService(application.Service)
}

type coreLifecycle interface {
	Start(context.Context) error
	Shutdown(context.Context) error
}

type wailsEventEmitter interface {
	Emit(string, ...any) bool
}

type wailsEventSink struct {
	events wailsEventEmitter
}

func newWailsEventSink(events wailsEventEmitter) *wailsEventSink {
	return &wailsEventSink{events: events}
}

func (sink *wailsEventSink) Emit(name string, payload any) error {
	if sink == nil || sink.events == nil {
		return errors.New("Wails event manager is unavailable")
	}
	sink.events.Emit(name, payload)
	return nil
}

// wailsLifecycleService adapts framework lifecycle callbacks to the unbound
// core. Its method names are Wails lifecycle hooks, not authored bridge calls.
type wailsLifecycleService struct {
	core coreLifecycle
}

func newWailsLifecycleService(core coreLifecycle) *wailsLifecycleService {
	return &wailsLifecycleService{core: core}
}

func (service *wailsLifecycleService) ServiceStartup(ctx context.Context, _ application.ServiceOptions) error {
	// Application-owned failures are recorded by the core in RuntimeStatus and
	// leave the master window available to explain the failure. Returning them
	// here would make Wails abort before that status can be presented.
	_ = service.core.Start(ctx)
	return nil
}

func (service *wailsLifecycleService) ServiceShutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), wailsShutdownTimeout)
	defer cancel()
	return service.core.Shutdown(ctx)
}

func registerWailsServices(host wailsServiceRegistrar, core *App) {
	host.RegisterService(application.NewService(newWailsLifecycleService(core)))
	host.RegisterService(application.NewService(newDesktopService(core)))
}
